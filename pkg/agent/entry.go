package agent

import (
	"context"
	"fmt"
	"net"
	"net/http"

	// for HTTP server runtime profiling
	_ "net/http/pprof"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/prometheus-auth/pkg/data"
	"github.com/rancher/prometheus-auth/pkg/kube"
	"github.com/rancher/steve/pkg/accesscontrol"
	"github.com/rancher/wrangler-api/pkg/generated/controllers/core"
	"github.com/rancher/wrangler-api/pkg/generated/controllers/rbac"

	"github.com/cockroachdb/cmux"
	"github.com/juju/errors"
	grpcproxy "github.com/mwitkow/grpc-proxy/proxy"
	promapi "github.com/prometheus/client_golang/api"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/net/netutil"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultThreadiness = 5
)

func Run(cliContext *cli.Context) {
	// enable profiler
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &agentConfig{
		ctx:                  ctx,
		listenAddress:        cliContext.String("listen-address"),
		monitoringNamespace:  cliContext.String("monitoring-namespace"),
		readTimeout:          cliContext.Duration("read-timeout"),
		maxConnections:       cliContext.Int("max-connections"),
		filterReaderLabelSet: data.NewSet(cliContext.StringSlice("filter-reader-labels")...),
	}

	proxyURLString := cliContext.String("proxy-url")
	if len(proxyURLString) == 0 {
		log.Fatal("--agent.proxy-url is blank")
	}
	proxyURL, err := url.Parse(proxyURLString)
	if err != nil {
		log.Fatal("Unable to parse agent.proxy-url")
	}
	cfg.proxyURL = proxyURL

	log.Println(cfg)

	reader, err := createAgent(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to create agent")
	}

	if err = reader.serve(); err != nil {
		log.WithError(err).Fatal("Failed to serve")
	}
}

type agentConfig struct {
	ctx                  context.Context
	listenAddress        string
	proxyURL             *url.URL
	readTimeout          time.Duration
	maxConnections       int
	filterReaderLabelSet data.Set
	monitoringNamespace  string
}

func (a *agentConfig) String() string {
	sb := &strings.Builder{}

	sb.WriteString(fmt.Sprint("listening on ", a.listenAddress))
	sb.WriteString(fmt.Sprint(", proxying to ", a.proxyURL.String()))
	sb.WriteString(fmt.Sprintf(" with ignoring 'remote reader' labels [%s]", a.filterReaderLabelSet))
	sb.WriteString(fmt.Sprintf(", only allow maximum %d connections with %v read timeout", a.maxConnections, a.readTimeout))
	sb.WriteString(" .")

	return sb.String()
}

type agent struct {
	cfg               *agentConfig
	listener          net.Listener
	nodes             kube.Nodes
	namespaces        kube.Namespaces
	secrets           *kube.Secrets
	remoteAPI         promapiv1.API
	controllerFactory controller.SharedControllerFactory
	myToken           string
}

func (a *agent) serve() error {
	//start controller
	if err := a.controllerFactory.Start(a.cfg.ctx, defaultThreadiness); err != nil {
		return err
	}

	listenerMux := cmux.New(a.listener)
	httpProxy := a.createHTTPProxy()
	grpcProxy := a.createGRPCProxy()

	errCh := make(chan error)
	go func() {
		if err := httpProxy.Serve(createHTTPListener(listenerMux)); err != nil {
			errCh <- errors.Annotate(err, "failed to start proxy http listener")
		}
	}()
	go func() {
		if err := grpcProxy.Serve(createGRPCListener(listenerMux)); err != nil {
			errCh <- errors.Annotate(err, "failed to start proxy grpc listener")
		}
	}()
	go func() {
		log.Infof("Start listening for connections on %s", a.cfg.listenAddress)

		if err := listenerMux.Serve(); err != nil {
			errCh <- errors.Annotatef(err, "failed to listen on %s", a.cfg.listenAddress)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-a.cfg.ctx.Done():
		grpcProxy.GracefulStop()
		httpProxy.Shutdown(a.cfg.ctx)
		return nil
	}
}

func createAgent(cfg *agentConfig) (*agent, error) {
	utilruntime.ReallyCrash = false
	utilruntime.PanicHandlers = []func(interface{}){
		func(i interface{}) {
			if err, ok := i.(error); ok {
				log.Error(errors.ErrorStack(err))
			} else {
				log.Error(i)
			}
		},
	}
	utilruntime.ErrorHandlers = []func(err error){
		func(err error) {
			log.Error(errors.ErrorStack(err))
		},
	}

	listener, err := net.Listen("tcp", cfg.listenAddress)
	if err != nil {
		return nil, errors.Annotatef(err, "unable to listen on addr %s", cfg.listenAddress)
	}
	listener = netutil.LimitListener(listener, cfg.maxConnections)

	// create Prometheus client
	promClient, err := promapi.NewClient(promapi.Config{
		Address: cfg.proxyURL.String(),
	})
	if err != nil {
		return nil, errors.Annotate(err, "unable to new Prometheus client")
	}

	k8sConfig, err := getKubeConfig()
	if err != nil {
		return nil, errors.Annotate(err, "unable to create Kubernetes config")
	}

	scheme := runtime.NewScheme()
	rbacv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	controllerFactory, err := controller.NewSharedControllerFactoryFromConfig(k8sConfig, scheme)
	if err != nil {
		return nil, err
	}

	coreClient := core.New(controllerFactory)
	rbacClient := rbac.New(controllerFactory)
	userAccessStore := accesscontrol.NewAccessStore(cfg.ctx, true, rbacClient.V1())

	nsIndexers := map[string]cache.IndexFunc{
		kube.ByProjectIDIndex: kube.NamespaceByProjectID,
	}
	secIndexers := map[string]cache.IndexFunc{
		kube.ByTokenIndex: kube.SecretByToken,
	}

	if err := coreClient.V1().Namespace().Informer().AddIndexers(nsIndexers); err != nil {
		return nil, errors.Annotate(err, "unable to add namespace indexer")
	}

	if err := coreClient.V1().Secret().Informer().AddIndexers(secIndexers); err != nil {
		return nil, errors.Annotate(err, "unable to add secret indexer")
	}
	secrets := kube.NewSecrets(cfg.ctx, coreClient.V1().Secret().Cache())
	return &agent{
		cfg:       cfg,
		listener:  listener,
		remoteAPI: promapiv1.NewAPI(promClient),
		nodes:     kube.NewNodes(cfg.ctx, userAccessStore),
		namespaces: kube.NewNamespaces(cfg.ctx, coreClient.V1().Namespace().Cache(), secrets,
			userAccessStore, cfg.monitoringNamespace),
		secrets:           secrets,
		controllerFactory: controllerFactory,
		myToken:           k8sConfig.BearerToken,
	}, nil
}

func (a *agent) createHTTPProxy() *http.Server {
	return &http.Server{
		Handler:     a.httpBackend(),
		ReadTimeout: a.cfg.readTimeout,
	}
}

func (a *agent) createGRPCProxy() *grpc.Server {
	return grpc.NewServer(
		grpc.CustomCodec(grpcproxy.Codec()),
		grpc.UnknownServiceHandler(a.grpcBackend()),
	)
}

func getKubeConfig() (*rest.Config, error) {
	kubeConfig := os.Getenv("KUBECONFIG")
	if kubeConfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeConfig)
	}

	return rest.InClusterConfig()
}

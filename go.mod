module github.com/rancher/prometheus-auth

go 1.13

replace (
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v36.1.0+incompatible
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible
	github.com/crewjam/saml => github.com/rancher/saml v0.0.0-20180713225824-ce1532152fde
	github.com/openzipkin-contrib/zipkin-go-opentracing => github.com/openzipkin-contrib/zipkin-go-opentracing v0.3.5
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v0.0.0-20200626085723-c448ada63d83
	github.com/rancher/prometheus-auth/pkg/data => ./pkg/data
	github.com/rancher/prometheus-auth/pkg/prom => ./pkg/prom
	github.com/rancher/steve => github.com/aiwantaozi/steve v0.0.0-20200726010056-fde154f84158
	k8s.io/client-go => k8s.io/client-go v0.18.0
)

require (
	github.com/cockroachdb/cmux v0.0.0-20170110192607-30d10be49292
	github.com/golang/protobuf v1.4.2
	github.com/golang/snappy v0.0.2-0.20190904063534-ff6b7dc882cf
	github.com/gorilla/mux v1.7.4
	github.com/hashicorp/go-hclog v0.14.0 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/json-iterator/go v1.1.9
	github.com/juju/errors v0.0.0-20200330140219-3fe23663418f
	github.com/juju/loggo v0.0.0-20180524022052-584905176618 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mwitkow/grpc-proxy v0.0.0-20181017164139-0f1106ef9c76
	github.com/onsi/ginkgo v1.13.0 // indirect
	github.com/prometheus/client_golang v1.6.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.10.0
	github.com/prometheus/prometheus v2.18.2+incompatible
	github.com/rancher/lasso v0.0.0-20200515155337-a34e1e26ad91
	github.com/rancher/norman v0.0.0-20200609220258-00d350370ee8 // indirect
	github.com/rancher/prometheus-auth/pkg/data v0.0.0
	github.com/rancher/prometheus-auth/pkg/prom v0.0.0
	github.com/rancher/steve v0.0.0-20200622175150-3dbc369174fb
	github.com/rancher/types v0.0.0-20200529180020-29fa023a5bd8
	github.com/rancher/wrangler-api v0.6.1-0.20200515193802-dcf70881b087
	github.com/sirupsen/logrus v1.4.2
	github.com/urfave/cli v1.22.2
	golang.org/x/net v0.0.0-20200822124328-c89045814202
	google.golang.org/grpc v1.31.1
	google.golang.org/grpc/examples v0.0.0-20200902210233-8630cac324bf // indirect
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce // indirect
	k8s.io/api v0.18.5
	k8s.io/apimachinery v0.18.5
	k8s.io/apiserver v0.18.5
	k8s.io/client-go v12.0.0+incompatible
)

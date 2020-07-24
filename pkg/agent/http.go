package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/rancher/prometheus-auth/pkg/data"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"k8s.io/apiserver/pkg/authentication/user"
)

func (a *agent) httpBackend() http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(a.cfg.proxyURL)
	router := mux.NewRouter()

	if log.GetLevel() == log.DebugLevel {
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
				log.Debugf("%s - %s", req.Method, req.URL.Path)

				next.ServeHTTP(resp, req)
			})
		})
	}

	// enable metrics
	router.Path("/_/metrics").Methods("GET").Handler(promhttp.Handler())

	// proxy white list
	router.Path("/alerts").Methods("GET").Handler(proxy)
	router.Path("/graph").Methods("GET").Handler(proxy)
	router.Path("/status").Methods("GET").Handler(proxy)
	router.Path("/flags").Methods("GET").Handler(proxy)
	router.Path("/config").Methods("GET").Handler(proxy)
	router.Path("/rules").Methods("GET").Handler(proxy)
	router.Path("/targets").Methods("GET").Handler(proxy)
	router.Path("/version").Methods("GET").Handler(proxy)
	router.Path("/service-discovery").Methods("GET").Handler(proxy)
	router.PathPrefix("/consoles/").Methods("GET").Handler(proxy)
	router.PathPrefix("/static/").Methods("GET").Handler(proxy)
	router.PathPrefix("/user/").Methods("GET").Handler(proxy)
	router.Path("/metrics").Methods("GET").Handler(proxy)
	router.Path("/-/healthy").Methods("GET").Handler(proxy)
	router.Path("/-/ready").Methods("GET").Handler(proxy)
	router.PathPrefix("/debug/").Methods("GET").Handler(proxy)

	// access control
	router.PathPrefix("/").Handler(accessControl(a, proxy))

	return router
}

func accessControl(agt *agent, proxyHandler http.Handler) http.Handler {
	router := mux.NewRouter()

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//user header
			rancherUser := r.Header.Get(rancherUserHeaderKey)
			rancherGroup := r.Header[rancherGroupHeaderKey]

			//sa header
			accessToken := strings.TrimPrefix(r.Header.Get(authorizationHeaderKey), "Bearer ")

			if rancherUser == "" && len(rancherGroup) == 0 && len(accessToken) == 0 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			var namespaceSet data.Set
			var info *user.DefaultInfo
			if rancherUser != "" || len(rancherGroup) != 0 {
				log.Debugf("%s - %s - access by userID", r.Method, r.URL.Path)
				info = &user.DefaultInfo{
					Name:   rancherUser,
					UID:    rancherUser,
					Groups: rancherGroup,
				}
			} else if len(accessToken) != 0 {
				log.Debugf("%s - %s - access by accessToken", r.Method, r.URL.Path)

				if agt.myToken == accessToken {
					proxyHandler.ServeHTTP(w, r)
					return
				}

				sa, err := agt.secrets.GetSA(accessToken)
				if err != nil {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}

				info = &user.DefaultInfo{
					Name: fmt.Sprintf("system:serviceaccount:%s:%s", sa.Namespace, sa.Name),
				}
			}

			if agt.nodes.CanList(info) {
				proxyHandler.ServeHTTP(w, r)
				return
			}

			namespaceSet = agt.namespaces.QueryByUser(info)

			apiCtx := &apiContext{
				tag:                  fmt.Sprintf("%016x", time.Now().Unix()),
				response:             w,
				request:              r,
				proxyHandler:         proxyHandler,
				filterReaderLabelSet: agt.cfg.filterReaderLabelSet,
				namespaceSet:         namespaceSet,
				remoteAPI:            agt.remoteAPI,
			}

			log.Debugf("common[%s] %s - %s can access namespaces %+v", apiCtx.tag, r.Method, r.URL.Path, apiCtx.namespaceSet.Values())

			newReqCtx := context.WithValue(r.Context(), apiContextKey, apiCtx)
			next.ServeHTTP(w, r.WithContext(newReqCtx))
		})
	})

	router.Path("/api/v1/query").Methods("GET", "POST").Handler(apiContextHandler(hijackQuery))
	router.Path("/api/v1/query_range").Methods("GET", "POST").Handler(apiContextHandler(hijackQueryRange))
	router.Path("/api/v1/series").Methods("GET").Handler(apiContextHandler(hijackSeries))
	router.Path("/api/v1/read").Methods("POST").Handler(apiContextHandler(hijackRead))
	router.Path("/api/v1/label/__name__/values").Methods("GET").Handler(apiContextHandler(hijackLabelName))
	router.Path("/api/v1/label/namespace/values").Methods("GET").Handler(apiContextHandler(hijackLabelNamespaces))
	router.Path("/api/v1/label/{name}/values").Methods("GET").Handler(proxyHandler)
	router.Path("/federate").Methods("GET").Handler(apiContextHandler(hijackFederate))

	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})

	return router
}

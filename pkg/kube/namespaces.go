package kube

import (
	"context"

	"github.com/rancher/prometheus-auth/pkg/data"
	"github.com/rancher/prometheus-auth/pkg/kube/rbac"
	"github.com/rancher/steve/pkg/accesscontrol"
	v1 "github.com/rancher/types/apis/core/v1"
	corev1 "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"

	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	schema2 "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/apiserver/pkg/authentication/user"
)

const (
	ByTokenIndex      = "byToken"
	ByProjectIDIndex  = "byProjectID"
	annServiceAccount = "kubernetes.io/service-account.name"
)

type Namespaces interface {
	QueryByUser(info *user.DefaultInfo) data.Set
	QueryByToken(token string) data.Set
}

type namespaces struct {
	ctx                  context.Context
	monitoringNamespace  string
	namespaceCache       corev1.NamespaceCache
	secrets              *Secrets
	accessStore          accesscontrol.AccessSetLookup
	saAccessStore        *rbac.AccessStore
	reviewResultTTLCache *cache.LRUExpireCache
}

func NewNamespaces(ctx context.Context,
	namespaceCache corev1.NamespaceCache, secrets *Secrets,
	acl *accesscontrol.AccessStore, sass *rbac.AccessStore, monitoringNs string) Namespaces {
	return &namespaces{
		ctx:                  ctx,
		monitoringNamespace:  monitoringNs,
		namespaceCache:       namespaceCache,
		secrets:              secrets,
		accessStore:          acl,
		saAccessStore:        sass,
		reviewResultTTLCache: cache.NewLRUExpireCache(1024),
	}
}

func (n *namespaces) QueryByUser(info *user.DefaultInfo) data.Set {
	ret, err := n.queryByUser(info)
	if err != nil {
		log.Warnln("failed to query Namespaces", errors.ErrorStack(err))
	}

	return ret
}

func (n *namespaces) queryByUser(info *user.DefaultInfo) (data.Set, error) {
	ret := data.Set{}
	objs, err := n.namespaceCache.List(labels.NewSelector())
	if err != nil {
		return nil, err
	}

	accessControl := rbac.NewUserLookupAccess(info, n.accessStore)
	for _, v := range objs {

		if v.DeletionTimestamp != nil {
			continue
		}

		if accessControl.CanAccess(v1.NamespaceGroupVersionKind.Group, v1.NamespaceResource.Name, v.Name, v.Namespace) {
			if n.monitoringNamespace == v.Name &&
				!accessControl.CanAccess(v1.PodGroupVersionKind.Group, v1.PodResource.Name, "*", v.Namespace) {
				continue
			}
			ret[v.Name] = struct{}{}
		}
	}

	return ret, nil
}

func (n *namespaces) QueryByToken(token string) data.Set {
	ret, err := n.queryByToken(token)
	if err != nil {
		log.Warnln("failed to query Namespaces", errors.ErrorStack(err))
	}

	return ret
}

func (n *namespaces) queryByToken(token string) (data.Set, error) {
	sa, err := n.secrets.GetSA(token)
	if err != nil {
		return nil, err
	}

	accessSet := n.saAccessStore.AccessFor(sa)
	objs, err := n.namespaceCache.List(labels.NewSelector())
	if err != nil {
		return nil, err
	}

	gr := schema2.GroupResource{
		Group:    v1.NamespaceGroupVersionKind.Group,
		Resource: v1.NamespaceResource.Name,
	}

	ret := data.Set{}
	for _, v := range objs {
		if accessSet.Grants("get", gr, v.Name, v.Name) || accessSet.Grants("list", gr, v.Name, v.Name) {
			ret[v.Name] = struct{}{}
		}
	}
	return ret, nil
}

func toNamespace(obj interface{}) *k8scorev1.Namespace {
	ns, ok := obj.(*k8scorev1.Namespace)
	if !ok {
		return &k8scorev1.Namespace{}
	}

	return ns
}

func getProjectID(ns *k8scorev1.Namespace) (string, bool) {
	if ns != nil && ns.Labels != nil {
		projectID, exist := ns.Labels["field.cattle.io/projectId"]
		if exist {
			return projectID, true
		}
	}

	return "", false
}

func NamespaceByProjectID(obj interface{}) ([]string, error) {
	projectID, exist := getProjectID(toNamespace(obj))
	if exist {
		return []string{projectID}, nil
	}

	return []string{}, nil
}

func SecretByToken(obj interface{}) ([]string, error) {
	sec := toSecret(obj)
	if sec.Type == k8scorev1.SecretTypeServiceAccountToken {
		secretToken := sec.Data[k8scorev1.ServiceAccountTokenKey]
		if len(secretToken) != 0 {
			return []string{string(secretToken)}, nil
		}
	}

	return []string{}, nil
}

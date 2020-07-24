package kube

import (
	"context"

	"github.com/rancher/prometheus-auth/pkg/data"
	"github.com/rancher/steve/pkg/accesscontrol"
	v1 "github.com/rancher/types/apis/core/v1"
	corev1 "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"

	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/authentication/user"
)

const (
	ByTokenIndex      = "byToken"
	ByProjectIDIndex  = "byProjectID"
	annServiceAccount = "kubernetes.io/service-account.name"
)

type Namespaces interface {
	QueryByUser(info *user.DefaultInfo) data.Set
}

type namespaces struct {
	ctx                 context.Context
	monitoringNamespace string
	namespaceCache      corev1.NamespaceCache
	secrets             *Secrets
	accessStore         accesscontrol.AccessSetLookup
}

func NewNamespaces(ctx context.Context,
	namespaceCache corev1.NamespaceCache, secrets *Secrets,
	acl *accesscontrol.AccessStore, monitoringNs string) Namespaces {
	return &namespaces{
		ctx:                 ctx,
		monitoringNamespace: monitoringNs,
		namespaceCache:      namespaceCache,
		secrets:             secrets,
		accessStore:         acl,
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

	accessControl := NewUserLookupAccess(info, n.accessStore)
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

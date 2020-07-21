package kube

import (
	"context"

	"github.com/rancher/prometheus-auth/pkg/kube/rbac"
	"github.com/rancher/steve/pkg/accesscontrol"
	v1 "github.com/rancher/types/apis/core/v1"

	k8scorev1 "k8s.io/api/core/v1"
	schema2 "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/authentication/user"
)

type Nodes interface {
	CanList(info *user.DefaultInfo) bool
	SACanList(sa *k8scorev1.ServiceAccount) bool
}

type nodes struct {
	accessStore   accesscontrol.AccessSetLookup
	saAccessStore *rbac.AccessStore
}

func NewNodes(ctx context.Context, acl *accesscontrol.AccessStore, sas *rbac.AccessStore) Nodes {
	return &nodes{
		accessStore:   acl,
		saAccessStore: sas,
	}
}

func (n *nodes) CanList(info *user.DefaultInfo) bool {
	accessControl := rbac.NewUserLookupAccess(info, n.accessStore)
	return accessControl.CanDo("list", v1.NodeGroupVersionKind.Group, v1.NodeResource.Name, "", "")
}

func (n *nodes) SACanList(sa *k8scorev1.ServiceAccount) bool {
	accessSet := n.saAccessStore.AccessFor(sa)

	gr := schema2.GroupResource{
		Group:    v1.NodeGroupVersionKind.Group,
		Resource: v1.NodeResource.Name,
	}

	return accessSet.Grants("list", gr, "", "")
}

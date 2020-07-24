package kube

import (
	"context"

	"github.com/rancher/steve/pkg/accesscontrol"
	v1 "github.com/rancher/types/apis/core/v1"

	"k8s.io/apiserver/pkg/authentication/user"
)

type Nodes interface {
	CanList(info *user.DefaultInfo) bool
}

type nodes struct {
	accessStore accesscontrol.AccessSetLookup
}

func NewNodes(ctx context.Context, acl *accesscontrol.AccessStore) Nodes {
	return &nodes{
		accessStore: acl,
	}
}

func (n *nodes) CanList(info *user.DefaultInfo) bool {
	accessControl := NewUserLookupAccess(info, n.accessStore)
	return accessControl.CanDo("list", v1.NodeGroupVersionKind.Group, v1.NodeResource.Name, "", "")
}

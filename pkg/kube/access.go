package kube

import (
	"github.com/rancher/steve/pkg/accesscontrol"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/authentication/user"
)

func NewUserLookupAccess(info *user.DefaultInfo, accessStore accesscontrol.AccessSetLookup) *UserCachedAccess {
	accessSet := accessStore.AccessFor(info)
	return &UserCachedAccess{
		access: accessSet,
	}
}

type UserCachedAccess struct {
	access *accesscontrol.AccessSet
}

func (a *UserCachedAccess) CanAccess(apiGroup, resource, name, ns string) bool {
	return a.CanDo("get", apiGroup, resource, name, ns) || a.CanDo("list", apiGroup, resource, name, ns)
}

func (a *UserCachedAccess) CanDo(verb, apiGroup, resource, name, ns string) bool {
	gr := schema.GroupResource{
		Group:    apiGroup,
		Resource: resource,
	}

	return a.access.Grants(verb, gr, ns, name)
}

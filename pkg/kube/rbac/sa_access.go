package rbac

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/rancher/steve/pkg/accesscontrol"
	v1 "github.com/rancher/wrangler-api/pkg/generated/controllers/rbac/v1"
	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/cache"
)

type AccessStore struct {
	serviceAccounts *policyRuleIndexer
	cache           *cache.LRUExpireCache
}

type roleKey struct {
	namespace string
	name      string
}

func NewSAAccessStore(ctx context.Context, cacheResults bool, rbac v1.Interface) *AccessStore {
	revisions := newRoleRevision(ctx, rbac)
	as := &AccessStore{
		serviceAccounts: newSAPolicyRuleIndexer(revisions, rbac),
	}
	if cacheResults {
		as.cache = cache.NewLRUExpireCache(50)
	}
	return as
}

func (l *AccessStore) AccessFor(sa *k8scorev1.ServiceAccount) *accesscontrol.AccessSet {
	var cacheKey string
	if l.cache != nil {
		cacheKey = l.CacheKey(sa)
		val, ok := l.cache.Get(cacheKey)
		if ok {
			as, _ := val.(*accesscontrol.AccessSet)
			return as
		}
	}

	result := l.serviceAccounts.get(sa.GetName())

	if l.cache != nil {
		result.ID = cacheKey
		l.cache.Add(cacheKey, result, 24*time.Hour)
	}

	return result
}

func (l *AccessStore) CacheKey(sa *k8scorev1.ServiceAccount) string {
	d := sha256.New()

	l.serviceAccounts.addRolesToHash(d, sa.GetName())

	return hex.EncodeToString(d.Sum(nil))
}

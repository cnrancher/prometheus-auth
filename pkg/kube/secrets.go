package kube

import (
	"context"

	corev1 "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"

	"github.com/juju/errors"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/cache"
)

type Secrets struct {
	ctx                  context.Context
	secretCache          corev1.SecretCache
	reviewResultTTLCache *cache.LRUExpireCache
}

func NewSecrets(ctx context.Context, secretCache corev1.SecretCache) *Secrets {
	return &Secrets{
		ctx:                  ctx,
		secretCache:          secretCache,
		reviewResultTTLCache: cache.NewLRUExpireCache(1024),
	}
}

func (n *Secrets) GetSA(token string) (*k8scorev1.ServiceAccount, error) {
	secList, err := n.secretCache.GetByIndex(ByTokenIndex, token)
	if err != nil || len(secList) != 1 {
		return nil, errors.Annotatef(err, "unknown token")
	}

	sec := secList[0]
	if sec.DeletionTimestamp != nil {
		return nil, errors.New("deleting token")
	}

	sa := &k8scorev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: sec.Namespace,
		},
	}

	saNameFromCache, exist := n.reviewResultTTLCache.Get(token)
	if exist {
		sa.Name = saNameFromCache.(string)
		return sa, nil
	}

	saName := sec.Annotations[annServiceAccount]
	if saName == "" {
		return nil, errors.New("failed to get serviceAccount for token " + token)
	}

	sa.Name = saName
	return sa, nil
}

func toSecret(obj interface{}) *k8scorev1.Secret {
	sec, ok := obj.(*k8scorev1.Secret)
	if !ok {
		return &k8scorev1.Secret{}
	}

	return sec
}

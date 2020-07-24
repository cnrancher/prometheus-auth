package kube

import (
	"context"

	corev1 "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"

	"github.com/juju/errors"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Secrets struct {
	ctx         context.Context
	secretCache corev1.SecretCache
}

func NewSecrets(ctx context.Context, secretCache corev1.SecretCache) *Secrets {
	return &Secrets{
		ctx:         ctx,
		secretCache: secretCache,
	}
}

func (n *Secrets) GetSA(token string) (*k8scorev1.ServiceAccount, error) {
	secList, err := n.secretCache.GetByIndex(ByTokenIndex, token)
	if err != nil {
		return nil, errors.Annotatef(err, "unknown token")
	}

	if len(secList) != 1 {
		return nil, errors.Annotatef(err, "can't find service account for token")
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

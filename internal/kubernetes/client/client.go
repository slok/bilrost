package client

import (
	"context"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	kubernetesbilrost "github.com/slok/bilrost/pkg/kubernetes/gen/clientset/versioned"
)

// Factory knows how to get new kubernetes clients.
type Factory interface {
	// NewCoreClient returns a Kubernetes client.
	NewCoreClient(ctx context.Context, cfg *rest.Config) (kubernetes.Interface, error)
	// NewCoreClient returns a Kubernetes client or Bilrost CRDs.
	NewBilrostClient(ctx context.Context, cfg *rest.Config) (kubernetesbilrost.Interface, error)
}

// BaseFactory is the base factory that knows how to return K8s clients.
const BaseFactory = baseFactory(0)

type baseFactory int

func (baseFactory) NewCoreClient(_ context.Context, cfg *rest.Config) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(cfg)
}

func (baseFactory) NewBilrostClient(_ context.Context, cfg *rest.Config) (kubernetesbilrost.Interface, error) {
	return kubernetesbilrost.NewForConfig(cfg)
}

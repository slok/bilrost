package controller

import (
	"context"

	"github.com/spotahome/kooper/controller"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	authv1 "github.com/slok/bilrost/pkg/apis/auth/v1"
)

// RetrieverKubernetesRepository is the service to manage k8s resources by the Kubernetes retrievers.
type RetrieverKubernetesRepository interface {
	ListIngresses(ctx context.Context, ns string, labelSelector map[string]string) (*networkingv1beta1.IngressList, error)
	WatchIngresses(ctx context.Context, ns string, labelSelector map[string]string) (watch.Interface, error)
	ListIngressAuths(ctx context.Context, ns string, labelSelector map[string]string) (*authv1.IngressAuthList, error)
	WatchIngressAuths(ctx context.Context, ns string, labelSelector map[string]string) (watch.Interface, error)
}

//go:generate mockery -case underscore -output controllermock -outpkg controllermock -name RetrieverKubernetesRepository

// NewIngressRetriever returns the retriever for ingress events.
func NewIngressRetriever(ns string, kuberepo RetrieverKubernetesRepository) controller.Retriever {
	return controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return kuberepo.ListIngresses(context.TODO(), ns, map[string]string{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return kuberepo.WatchIngresses(context.TODO(), ns, map[string]string{})
		},
	})
}

// NewIngressAuthRetriever returns the retriever for ingress auth CR events.
func NewIngressAuthRetriever(ns string, kuberepo RetrieverKubernetesRepository) controller.Retriever {
	return controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return kuberepo.ListIngressAuths(context.TODO(), ns, map[string]string{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return kuberepo.WatchIngressAuths(context.TODO(), ns, map[string]string{})
		},
	})
}

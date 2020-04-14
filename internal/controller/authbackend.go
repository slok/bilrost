package controller

import (
	"context"

	"github.com/spotahome/kooper/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/slok/bilrost/internal/log"
	authv1 "github.com/slok/bilrost/pkg/apis/auth/v1"
	kubernetesbilrost "github.com/slok/bilrost/pkg/kubernetes/gen/clientset/versioned"
)

// NewAuthBackendRetriever returns the retriever for the Auth backend controller.
func NewAuthBackendRetriever(cli kubernetesbilrost.Interface) controller.Retriever {
	return controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return cli.AuthV1().AuthBackends().List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return cli.AuthV1().AuthBackends().Watch(options)
		},
	})
}

// NewAuthBackendHandler returns the handler for the Auth backend controller.
func NewAuthBackendHandler(logger log.Logger) controller.Handler {
	logger = logger.WithKV(log.KV{"service": "controller.AuthBackendHandler"})

	return controller.HandlerFunc(func(ctx context.Context, obj runtime.Object) error {
		ab, ok := obj.(*authv1.AuthBackend)
		if !ok {
			logger.Debugf("kubernetes received object is not an AuthBackend")
			return nil
		}

		logger = logger.WithKV(log.KV{"obj-id": ab.Name})
		logger.Infof("handling object...")

		return nil
	})
}

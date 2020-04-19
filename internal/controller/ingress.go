package controller

import (
	"context"
	"fmt"

	"github.com/spotahome/kooper/controller"
	networkingv1beta "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/model"
	"github.com/slok/bilrost/internal/security"
)

// IngressControllerKubeService is the service to manage k8s resources by the ingress
// controller.
type IngressControllerKubeService interface {
	ListIngresses(ctx context.Context, ns string, labelSelector map[string]string) (*networkingv1beta.IngressList, error)
	WatchIngresses(ctx context.Context, ns string, labelSelector map[string]string) (watch.Interface, error)
}

// NewIngressRetriever returns the retriever for the ingress controller.
func NewIngressRetriever(ns string, kubeSvc IngressControllerKubeService) controller.Retriever {
	return controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return kubeSvc.ListIngresses(context.TODO(), ns, map[string]string{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return kubeSvc.WatchIngresses(context.TODO(), ns, map[string]string{})
		},
	})
}

// IngressHandlerConfig is the configuration of the ingress handler.
type IngressHandlerConfig struct {
	SecuritySvc              security.Service
	IngressBackendAnnotation string
	Logger                   log.Logger
}

type ingressHandler struct {
	annotation string

	securitySvc security.Service
	logger      log.Logger
}

func (c *IngressHandlerConfig) defaults() error {
	if c.Logger == nil {
		c.Logger = log.Dummy
	}
	c.Logger = c.Logger.WithKV(log.KV{"service": "controller.IngressHandler"})

	if c.IngressBackendAnnotation == "" {
		c.IngressBackendAnnotation = "auth.bilrost.slok.dev/backend"
	}

	if c.SecuritySvc == nil {
		return fmt.Errorf("security service is required")
	}

	return nil
}

// NewIngressHandler returns the handler for the ingress controller.
func NewIngressHandler(cfg IngressHandlerConfig) (controller.Handler, error) {
	err := cfg.defaults()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return ingressHandler{
		annotation:  cfg.IngressBackendAnnotation,
		securitySvc: cfg.SecuritySvc,
		logger:      cfg.Logger,
	}, nil
}

func (i ingressHandler) Handle(ctx context.Context, obj runtime.Object) error {
	ing, ok := obj.(*networkingv1beta.Ingress)
	if !ok {
		i.logger.Debugf("kubernetes received object is not an Ingress")
		return nil
	}

	logger := i.logger.WithKV(log.KV{"obj-ns": ing.Namespace, "obj-id": ing.Name})

	backendID, ok := ing.Annotations[i.annotation]
	if !ok {
		logger.Debugf("ignoring ingress...")
		return nil
	}

	logger.Infof("handling ingress...")

	rulesLen := len(ing.Spec.Rules)
	if rulesLen != 1 {
		return fmt.Errorf("required rules on ingress is 1, got %d", rulesLen)
	}

	pathsLen := len(ing.Spec.Rules[0].HTTP.Paths)
	if pathsLen != 1 {
		return fmt.Errorf("required paths on ingress is 1, got %d", pathsLen)
	}

	app := model.App{
		ID:            fmt.Sprintf("%s/%s", ing.Namespace, ing.Name),
		AuthBackendID: backendID,
		Host:          ing.Spec.Rules[0].Host,
		Ingress: model.KubernetesIngress{
			Name:      ing.Name,
			Namespace: ing.Namespace,
			Upstream: model.KubernetesService{
				Name:           ing.Spec.Rules[0].HTTP.Paths[0].Backend.ServiceName,
				Namespace:      ing.Namespace,
				PortOrPortName: ing.Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort.String(),
			},
		},
	}
	err := i.securitySvc.SecureApp(ctx, app)
	if err != nil {
		return fmt.Errorf("could not secure the application: %w", err)
	}

	return nil
}

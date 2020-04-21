package controller

import (
	"context"
	"fmt"

	"github.com/spotahome/kooper/controller"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/model"
	"github.com/slok/bilrost/internal/security"
)

const (
	backendAnnotation = "auth.bilrost.slok.dev/backend"
	handledAnnotation = "auth.bilrost.slok.dev/handled"
)

// IngressControllerKubeService is the service to manage k8s resources by the ingress
// controller.
type IngressControllerKubeService interface {
	ListIngresses(ctx context.Context, ns string, labelSelector map[string]string) (*networkingv1beta1.IngressList, error)
	WatchIngresses(ctx context.Context, ns string, labelSelector map[string]string) (watch.Interface, error)
	GetIngress(ctx context.Context, ns, name string) (*networkingv1beta1.Ingress, error)
	UpdateIngress(ctx context.Context, ingress *networkingv1beta1.Ingress) error
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
	KubeSvc     IngressControllerKubeService
	SecuritySvc security.Service
	Logger      log.Logger
}

type ingressHandler struct {
	kubeSvc     IngressControllerKubeService
	securitySvc security.Service
	logger      log.Logger
}

func (c *IngressHandlerConfig) defaults() error {
	if c.Logger == nil {
		c.Logger = log.Dummy
	}
	c.Logger = c.Logger.WithKV(log.KV{"service": "controller.IngressHandler"})

	if c.KubeSvc == nil {
		return fmt.Errorf("kubernetes service is required")
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
		kubeSvc:     cfg.KubeSvc,
		securitySvc: cfg.SecuritySvc,
		logger:      cfg.Logger,
	}, nil
}

func (i ingressHandler) Handle(ctx context.Context, obj runtime.Object) error {
	ing, ok := obj.(*networkingv1beta1.Ingress)
	if !ok {
		i.logger.Debugf("kubernetes received object is not an Ingress")
		return nil
	}

	logger := i.logger.WithKV(log.KV{"obj-ns": ing.Namespace, "obj-id": ing.Name})

	rulesLen := len(ing.Spec.Rules)
	if rulesLen != 1 {
		return fmt.Errorf("required rules on ingress is 1, got %d", rulesLen)
	}

	pathsLen := len(ing.Spec.Rules[0].HTTP.Paths)
	if pathsLen != 1 {
		return fmt.Errorf("required paths on ingress is 1, got %d", pathsLen)
	}

	backendID, hasBackend := ing.Annotations[backendAnnotation]
	_, prevHandled := ing.Annotations[handledAnnotation]

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

	// Select the correct action for the ingress.
	switch {
	// If we have a backend, then we need to trigger securing process.
	case hasBackend && backendID != "":
		logger.Infof("start securing ingress...")
		err := i.securitySvc.SecureApp(ctx, app)
		if err != nil {
			return fmt.Errorf("could not secure the application: %w", err)
		}

		// Mark the ingress as being handled so we can rollback although the user
		// removes the annotation.
		err = i.markIngress(ctx, ing.Namespace, ing.Name)
		if err != nil {
			return fmt.Errorf("could not mark the ingress as handled: %w", err)
		}

		return nil

	// If we don't have backend but we have the ingress marked means that we
	// need to trigger a clean up process.
	case (!hasBackend || backendID == "") && prevHandled:
		logger.Infof("start rollbacking ingress security...")
		err := i.securitySvc.RollbackAppSecurity(ctx, app)
		if err != nil {
			return fmt.Errorf("could not rollback the ingress security: %w", err)
		}

		// Not ours anymore, remove the habdled mark.
		err = i.unmarkIngress(ctx, ing.Namespace, ing.Name)
		if err != nil {
			return fmt.Errorf("could not mark the ingress as handled: %w", err)
		}

		return nil
	}

	logger.Debugf("ignoring ingress...")

	return nil
}

func (i ingressHandler) markIngress(ctx context.Context, ns, name string) error {
	storedIng, err := i.kubeSvc.GetIngress(ctx, ns, name)
	if err != nil {
		return fmt.Errorf("could not get ingress: %w", err)
	}

	// If already marked, skip update.
	if _, ok := storedIng.Annotations[handledAnnotation]; ok {
		return nil
	}

	storedIng.Annotations[handledAnnotation] = "true"
	err = i.kubeSvc.UpdateIngress(ctx, storedIng)
	if err != nil {
		return fmt.Errorf("could not update ingress: %w", err)
	}

	return nil
}

func (i ingressHandler) unmarkIngress(ctx context.Context, ns, name string) error {
	storedIng, err := i.kubeSvc.GetIngress(ctx, ns, name)
	if err != nil {
		return fmt.Errorf("could not get ingress: %w", err)
	}

	// If not marked, skip update.
	if _, ok := storedIng.Annotations[handledAnnotation]; !ok {
		return nil
	}

	delete(storedIng.Annotations, handledAnnotation)
	err = i.kubeSvc.UpdateIngress(ctx, storedIng)
	if err != nil {
		return fmt.Errorf("could not update ingress: %w", err)
	}

	return nil
}

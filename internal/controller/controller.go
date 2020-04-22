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

// KubernetesRepository is the service to manage k8s resources by the Kubernetes controller.
type KubernetesRepository interface {
	ListIngresses(ctx context.Context, ns string, labelSelector map[string]string) (*networkingv1beta1.IngressList, error)
	WatchIngresses(ctx context.Context, ns string, labelSelector map[string]string) (watch.Interface, error)
	GetIngress(ctx context.Context, ns, name string) (*networkingv1beta1.Ingress, error)
	UpdateIngress(ctx context.Context, ingress *networkingv1beta1.Ingress) error
}

//go:generate mockery -case underscore -output controllermock -outpkg controllermock -name KubernetesRepository

// NewRetriever returns the retriever for the controller.
func NewRetriever(ns string, kuberepo KubernetesRepository) controller.Retriever {
	return controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return kuberepo.ListIngresses(context.TODO(), ns, map[string]string{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return kuberepo.WatchIngresses(context.TODO(), ns, map[string]string{})
		},
	})
}

// HandlerConfig is the configuration of the controller handler.
type HandlerConfig struct {
	KubernetesRepo KubernetesRepository
	SecuritySvc    security.Service
	Logger         log.Logger
}

func (c *HandlerConfig) defaults() error {
	if c.Logger == nil {
		c.Logger = log.Dummy
	}
	c.Logger = c.Logger.WithKV(log.KV{"service": "controller.Handler"})

	if c.KubernetesRepo == nil {
		return fmt.Errorf("kubernetes repository is required")
	}

	if c.SecuritySvc == nil {
		return fmt.Errorf("security service is required")
	}

	return nil
}

type handler struct {
	repo        KubernetesRepository
	securitySvc security.Service
	logger      log.Logger
}

// NewHandler returns the handler for the controller.
func NewHandler(cfg HandlerConfig) (controller.Handler, error) {
	err := cfg.defaults()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return handler{
		repo:        cfg.KubernetesRepo,
		securitySvc: cfg.SecuritySvc,
		logger:      cfg.Logger,
	}, nil
}

func (h handler) Handle(ctx context.Context, obj runtime.Object) error {
	ing, ok := obj.(*networkingv1beta1.Ingress)
	if !ok {
		h.logger.Debugf("kubernetes received object is not an Ingress")
		return nil
	}

	logger := h.logger.WithKV(log.KV{"obj-ns": ing.Namespace, "obj-id": ing.Name})

	// Should we ignore the ingress?
	backendID := ing.Annotations[backendAnnotation]
	_, prevHandled := ing.Annotations[handledAnnotation]
	if backendID == "" && !prevHandled {
		logger.Debugf("ignoring ingress...")
		return nil
	}

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

	// Select the correct action for the ingress.
	switch {
	// If we have a backend, then we need to trigger securing process.
	case backendID != "":
		logger.Infof("start securing ingress...")
		err := h.securitySvc.SecureApp(ctx, app)
		if err != nil {
			return fmt.Errorf("could not secure the application: %w", err)
		}

		// Mark the ingress as being handled so we can rollback although the user
		// removes the annotation.
		err = h.markIngress(ctx, ing.Namespace, ing.Name)
		if err != nil {
			return fmt.Errorf("could not mark the ingress as handled: %w", err)
		}

		return nil

	// If we don't have backend but we have the ingress marked means that we
	// need to trigger a clean up process.
	case backendID == "" && prevHandled:
		logger.Infof("start rollbacking ingress security...")
		err := h.securitySvc.RollbackAppSecurity(ctx, app)
		if err != nil {
			return fmt.Errorf("could not rollback the ingress security: %w", err)
		}

		// Not ours anymore, remove the habdled mark.
		err = h.unmarkIngress(ctx, ing.Namespace, ing.Name)
		if err != nil {
			return fmt.Errorf("could not mark the ingress as handled: %w", err)
		}

		return nil
	}

	logger.Debugf("ignoring ingress...")

	return nil
}

func (h handler) markIngress(ctx context.Context, ns, name string) error {
	storedIng, err := h.repo.GetIngress(ctx, ns, name)
	if err != nil {
		return fmt.Errorf("could not get ingress: %w", err)
	}

	// If already marked, skip update.
	if _, ok := storedIng.Annotations[handledAnnotation]; ok {
		return nil
	}

	storedIng.Annotations[handledAnnotation] = "true"
	err = h.repo.UpdateIngress(ctx, storedIng)
	if err != nil {
		return fmt.Errorf("could not update ingress: %w", err)
	}

	return nil
}

func (h handler) unmarkIngress(ctx context.Context, ns, name string) error {
	storedIng, err := h.repo.GetIngress(ctx, ns, name)
	if err != nil {
		return fmt.Errorf("could not get ingress: %w", err)
	}

	// If not marked, skip update.
	if _, ok := storedIng.Annotations[handledAnnotation]; !ok {
		return nil
	}

	delete(storedIng.Annotations, handledAnnotation)
	err = h.repo.UpdateIngress(ctx, storedIng)
	if err != nil {
		return fmt.Errorf("could not update ingress: %w", err)
	}

	return nil
}

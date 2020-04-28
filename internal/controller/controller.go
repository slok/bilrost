package controller

import (
	"context"
	"fmt"

	"github.com/spotahome/kooper/controller"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/slok/bilrost/internal/kubernetes/labels"
	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/model"
	"github.com/slok/bilrost/internal/security"
)

const (
	backendAnnotation = "auth.bilrost.slok.dev/backend"
	handledAnnotation = "auth.bilrost.slok.dev/handled"
	securityfinalizer = "finalizers.auth.bilrost.slok.dev/security"
)

// HandlerKubernetesRepository is the service to manage k8s resources by the Kubernetes controller.
type HandlerKubernetesRepository interface {
	GetIngress(ctx context.Context, ns, name string) (*networkingv1beta1.Ingress, error)
	UpdateIngress(ctx context.Context, ingress *networkingv1beta1.Ingress) error
}

//go:generate mockery -case underscore -output controllermock -outpkg controllermock -name HandlerKubernetesRepository

// HandlerConfig is the configuration of the controller handler.
type HandlerConfig struct {
	KubernetesRepo HandlerKubernetesRepository
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
	repo        HandlerKubernetesRepository
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

	var ing *networkingv1beta1.Ingress
	switch v := obj.(type) {
	case *corev1.Secret, *corev1.Service:
		// Generated types, we will get the ingress data form the convention labels of
		// the generated resource and get the original ingress to trigger the reconciliation
		// loop in case any of the generated resource have changed.
		vo, ok := v.(metav1.Object)
		if !ok {
			h.logger.Debugf("kubernetes received object is not a valid type")
			return nil
		}

		h.logger.Infof("triggered by %s/%s", vo.GetNamespace(), vo.GetName())

		// Get original ingress data form the tracked label.
		ns, name, err := labels.DecodeSourceLabelValue(vo.GetLabels()[labels.LabelKeySource])
		if err != nil {
			h.logger.Warningf("could not get original resource data from generated resource labels: %s", err)
			return nil
		}

		ing, err = h.repo.GetIngress(ctx, ns, name)
		if err != nil {
			return fmt.Errorf("could not get ingress: %w", err)
		}

	case *networkingv1beta1.Ingress:
		// Regular main reconciliation use case.
		ing = v
	default:
		// Unknown type, skip.
		h.logger.Debugf("kubernetes received object is not a valid type")
		return nil
	}

	// Start reconciliation loop.
	err := h.handleIngress(ctx, ing)
	if err != nil {
		return err
	}

	return nil
}

func (h handler) handleIngress(ctx context.Context, ing *networkingv1beta1.Ingress) error {
	logger := h.logger.WithKV(log.KV{"obj-ns": ing.Namespace, "obj-id": ing.Name})

	// Get the possible states of an ingress.
	wantHandle := ing.Annotations[backendAnnotation] != ""
	_, readyToBeHandled := ing.Annotations[handledAnnotation]
	wantDelete := !ing.DeletionTimestamp.IsZero()
	clean := wantDelete && !sliceContainsString(ing.ObjectMeta.Finalizers, securityfinalizer)

	// check if we need to handle.
	if !wantHandle && !readyToBeHandled {
		logger.Debugf("ignoring ingress...")
		return nil
	}

	err := validateIngress(ing)
	if err != nil {
		return fmt.Errorf("the ingress that we want to handle is not valid: %w", err)
	}

	// Select the correct action for the ingress.
	switch {

	// If the ingress has been deleted and we already cleaned then skip.
	// Use case: The user has deleted the ingress and we	 handled the clean process.
	case clean:
		logger.Debugf("already clean, nothing to do here...")
		return nil

	// If the ingress is not ready to be handled then prepare for the handling
	// Use case: The user just added the backend annotation.
	case wantHandle && !readyToBeHandled && !wantDelete:
		// Set the required information on the ingress to be ready to start
		// the security reconciliation loop.
		err := h.ensureIngressReady(ctx, ing.Namespace, ing.Name)
		if err != nil {
			return fmt.Errorf("could not ensure the ingress ready to be handled: %w", err)
		}

		return nil

	// If we have a backend and we are ready to handle, then we need to trigger securing process.
	// Use case: The ingress is in a common handling state of reconciliation loop.
	case wantHandle && readyToBeHandled && !wantDelete:
		logger.Infof("start securing ingress...")

		err := h.securitySvc.SecureApp(ctx, mapToModel(ing))
		if err != nil {
			return fmt.Errorf("could not secure the application: %w", err)
		}

		// In case someone has deleted our internal marks (handled annotation, finalizers...)
		// ensure they are present.
		err = h.ensureIngressReady(ctx, ing.Namespace, ing.Name)
		if err != nil {
			return fmt.Errorf("could not ensure the ingress ready to be handled: %w", err)
		}

		return nil

	// If we don't have backend but we have the ingress marked means that we
	// need to trigger a clean up process. Or if the user has deleted the ingress.
	// Use case: The user has removed the backend annotation.
	// Use case: The user has deleted the ingress.
	case !wantHandle && readyToBeHandled, wantDelete:
		logger.Infof("start rollbacking ingress security...")

		err := h.securitySvc.RollbackAppSecurity(ctx, mapToModel(ing))
		if err != nil {
			return fmt.Errorf("could not rollback the ingress security: %w", err)
		}

		// Not ours anymore, remove the habdled mark.
		err = h.ensureIngressClean(ctx, ing.Namespace, ing.Name)
		if err != nil {
			return fmt.Errorf("could not mark the ingress as before (clean): %w", err)
		}

		return nil
	}

	return fmt.Errorf("we shouldn't reach here... use case not implemented")
}

func (h handler) ensureIngressReady(ctx context.Context, ns, name string) error {
	storedIng, err := h.repo.GetIngress(ctx, ns, name)
	if err != nil {
		return fmt.Errorf("could not get ingress: %w", err)
	}

	finalizerPresent := sliceContainsString(storedIng.ObjectMeta.Finalizers, securityfinalizer)
	_, handledAnnotPresent := storedIng.Annotations[handledAnnotation]

	// If the ingress already ready, then don't update.
	if finalizerPresent && handledAnnotPresent {
		return nil
	}

	// Set the information required on the ingress and update.
	storedIng.Annotations[handledAnnotation] = "true"
	if !finalizerPresent {
		storedIng.ObjectMeta.Finalizers = append(storedIng.ObjectMeta.Finalizers, securityfinalizer)
	}
	err = h.repo.UpdateIngress(ctx, storedIng)
	if err != nil {
		return fmt.Errorf("could not update ingress: %w", err)
	}

	return nil
}

func (h handler) ensureIngressClean(ctx context.Context, ns, name string) error {
	storedIng, err := h.repo.GetIngress(ctx, ns, name)
	if err != nil {
		return fmt.Errorf("could not get ingress: %w", err)
	}

	finalizerPresent := sliceContainsString(storedIng.ObjectMeta.Finalizers, securityfinalizer)
	_, handledAnnotPresent := storedIng.Annotations[handledAnnotation]

	// If the ingress already clean, then don't update.
	if !finalizerPresent && !handledAnnotPresent {
		return nil
	}

	// Set the information required on the ingress and update.
	delete(storedIng.Annotations, handledAnnotation)
	for i, f := range storedIng.ObjectMeta.Finalizers {
		if f == securityfinalizer {
			storedIng.ObjectMeta.Finalizers = append(storedIng.ObjectMeta.Finalizers[:i], storedIng.ObjectMeta.Finalizers[i+1:]...)
			break
		}
	}
	err = h.repo.UpdateIngress(ctx, storedIng)
	if err != nil {
		return fmt.Errorf("could not update ingress: %w", err)
	}

	return nil
}

func validateIngress(ing *networkingv1beta1.Ingress) error {
	rulesLen := len(ing.Spec.Rules)
	if rulesLen != 1 {
		return fmt.Errorf("required rules on ingress is 1, got %d", rulesLen)
	}

	pathsLen := len(ing.Spec.Rules[0].HTTP.Paths)
	if pathsLen != 1 {
		return fmt.Errorf("required paths on ingress is 1, got %d", pathsLen)
	}

	return nil
}

func mapToModel(ing *networkingv1beta1.Ingress) model.App {
	return model.App{
		ID:            fmt.Sprintf("%s/%s", ing.Namespace, ing.Name),
		AuthBackendID: ing.Annotations[backendAnnotation],
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
}

func sliceContainsString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

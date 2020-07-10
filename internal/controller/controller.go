package controller

import (
	"context"
	"fmt"

	"github.com/spotahome/kooper/v2/controller"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/security"
	authv1 "github.com/slok/bilrost/pkg/apis/auth/v1"
)

const (
	backendAnnotation = "auth.bilrost.slok.dev/backend"
	handledAnnotation = "auth.bilrost.slok.dev/handled"
	securityfinalizer = "finalizers.auth.bilrost.slok.dev/security"
)

// HandlerKubernetesRepository is the service to manage k8s resources by the Kubernetes handler.
type HandlerKubernetesRepository interface {
	GetIngressAuth(ctx context.Context, ns, name string) (*authv1.IngressAuth, error)
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
//
// This handler is the entrypoint of all the logic that is triggered with the controller pattern,
// this handle will be called based on the Kubernetes received events (subscribed using the retrievers)
// so this handler knows how to handle changes based on ingress obects and ingressAuth CR objects
// depending on what is the received updated object it will call internally the required
// handle process.
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

// Handle implements a kooper controller handler that will receive "change" events on Kubernetes resources
//
// The logic begind the handler is to act always with the same data independently of the
// object event is received. For example:
// - If an ingress resource event is received we will try getting the corresponding ingressAuth CR and execute handling logic.
// - If an ingressAuth CR is received we will try getting the ingress associated (same ns and name) and execute handling logic.
//
// In case we don't have an IngressAuth CR associated with the ingress, we will use the default data as a
// fallback, this gives us the ability to reconcile based only in ingress data.
// On the other side if we have IngressAuth CR we will use this IngressAuth data for the reconciliation.
// So we could put it in simple words: Ingress data is required, IngressAuth CR data is optional (used to set advanced options).
//
// TODO(slok): Not optimized in k8s resource calls, some calls are not required, check this when we have problems with it.
func (h handler) Handle(ctx context.Context, obj runtime.Object) error {
	switch v := obj.(type) {
	case *networkingv1beta1.Ingress:
		h.logger.Debugf("ingress event received...")
		return h.handle(ctx, v, nil)
	case *authv1.IngressAuth:
		h.logger.Debugf("ingressAuth event received...")
		// We need ingress information, if not present then error.
		ing, err := h.repo.GetIngress(ctx, v.Namespace, v.Name)
		if err != nil {
			return fmt.Errorf("ingress resource for %s/%s not found, required: %w", v.Namespace, v.Name, err)
		}

		return h.handle(ctx, ing, v)
	}

	h.logger.Debugf("kubernetes received object is not a valid type to be handled")
	return nil
}

func (h handler) handle(ctx context.Context, ing *networkingv1beta1.Ingress, ia *authv1.IngressAuth) error {
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

		// Try getting advanced options from the CR.
		if ia == nil {
			ia, err = h.tryGetIngressAuth(ctx, ing.Namespace, ing.Name)
			if err != nil {
				return err
			}
		}

		err := h.securitySvc.SecureApp(ctx, mapToModel(ing, ia))
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

		// Try getting advanced options from the CR.
		// TODO(slok): Do we need to get advanced options?
		if ia == nil {
			ia, err = h.tryGetIngressAuth(ctx, ing.Name, ing.Namespace)
			if err != nil {
				return err
			}

		}

		err := h.securitySvc.RollbackAppSecurity(ctx, mapToModel(ing, ia))
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

// tryGetIngressAuth if not present returns nil.
func (h handler) tryGetIngressAuth(ctx context.Context, namespace, name string) (*authv1.IngressAuth, error) {
	ai, err := h.repo.GetIngressAuth(ctx, namespace, name)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get ingress auth: %w", err)
	}

	return ai, nil
}

func sliceContainsString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

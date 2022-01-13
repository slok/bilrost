package kubernetes

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/slok/bilrost/internal/metrics"
	"github.com/slok/bilrost/internal/model"
	authv1 "github.com/slok/bilrost/pkg/apis/auth/v1"
)

// MeasuredService is like Service but measuring with a metrics.Recorder
// all the operations made.
type MeasuredService struct {
	rec  metrics.Recorder
	next Service
}

// NewMeasuredService wraps a kubernetes.Service to measure using a metrics.Recorder.
func NewMeasuredService(rec metrics.Recorder, next Service) MeasuredService {
	return MeasuredService{rec: rec, next: next}
}

// GetAuthBackend satisifies controller.AuthBackendRepository interface.
func (m MeasuredService) GetAuthBackend(ctx context.Context, id string) (a *model.AuthBackend, err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, "", "GetAuthBackend", err == nil, t0)
	}(time.Now())
	return m.next.GetAuthBackend(ctx, id)
}

// GetIngressAuth satisfies multiple interfaces.
func (m MeasuredService) GetIngressAuth(ctx context.Context, namespace, name string) (ia *authv1.IngressAuth, err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, namespace, "GetIngressAuth", err == nil, t0)
	}(time.Now())
	return m.next.GetIngressAuth(ctx, namespace, name)
}

// ListIngressAuths satisfies multiple interfaces.
func (m MeasuredService) ListIngressAuths(ctx context.Context, namespace string, labelSelector map[string]string) (ial *authv1.IngressAuthList, err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, namespace, "ListIngressAuths", err == nil, t0)
	}(time.Now())
	return m.next.ListIngressAuths(ctx, namespace, labelSelector)
}

// WatchIngressAuths satisfies multiple interfaces.
func (m MeasuredService) WatchIngressAuths(ctx context.Context, namespace string, labelSelector map[string]string) (i watch.Interface, err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, namespace, "WatchIngressAuths", err == nil, t0)
	}(time.Now())
	return m.next.WatchIngressAuths(ctx, namespace, labelSelector)
}

// EnsureDeployment satisfies oauth2proxy.KubernetesRepository interface.
func (m MeasuredService) EnsureDeployment(ctx context.Context, dep *appsv1.Deployment) (err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, dep.Namespace, "EnsureDeployment", err == nil, t0)
	}(time.Now())
	return m.next.EnsureDeployment(ctx, dep)
}

// DeleteDeployment satisfies oauth2proxy.KubernetesRepository interface.
func (m MeasuredService) DeleteDeployment(ctx context.Context, ns, name string) (err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, ns, "DeleteDeployment", err == nil, t0)
	}(time.Now())
	return m.next.DeleteDeployment(ctx, ns, name)
}

// EnsureService satisfies oauth2proxy.KubernetesRepository interface.
func (m MeasuredService) EnsureService(ctx context.Context, svc *corev1.Service) (err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, svc.Namespace, "EnsureService", err == nil, t0)
	}(time.Now())
	return m.next.EnsureService(ctx, svc)
}

// DeleteService satisfies oauth2proxy.KubernetesRepository interface.
func (m MeasuredService) DeleteService(ctx context.Context, ns, name string) (err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, ns, "DeleteService", err == nil, t0)
	}(time.Now())
	return m.next.DeleteService(ctx, ns, name)
}

// GetSecret satisfies dex.KubernetesRepository interface.
func (m MeasuredService) GetSecret(ctx context.Context, ns, name string) (s *corev1.Secret, err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, ns, "GetSecret", err == nil, t0)
	}(time.Now())
	return m.next.GetSecret(ctx, ns, name)
}

// EnsureSecret satisfies oauth2proxy.KubernetesRepository interface.
func (m MeasuredService) EnsureSecret(ctx context.Context, secret *corev1.Secret) (err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, secret.Namespace, "EnsureSecret", err == nil, t0)
	}(time.Now())
	return m.next.EnsureSecret(ctx, secret)
}

// DeleteSecret satisfies oauth2proxy.KubernetesRepository interface.
func (m MeasuredService) DeleteSecret(ctx context.Context, ns, name string) (err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, ns, "DeleteSecret", err == nil, t0)
	}(time.Now())
	return m.next.DeleteSecret(ctx, ns, name)
}

// GetIngress satisfies oauth2proxy.KubernetesRepository interface.
func (m MeasuredService) GetIngress(ctx context.Context, ns, name string) (i *networkingv1beta1.Ingress, err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, ns, "GetIngress", err == nil, t0)
	}(time.Now())
	return m.next.GetIngress(ctx, ns, name)
}

// UpdateIngress satisfies oauth2proxy.KubernetesRepository interface.
func (m MeasuredService) UpdateIngress(ctx context.Context, ingress *networkingv1beta1.Ingress) (err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, ingress.Namespace, "UpdateIngress", err == nil, t0)
	}(time.Now())
	return m.next.UpdateIngress(ctx, ingress)
}

// ListIngresses satisfies controller.IngressControllerKubeService interface.
func (m MeasuredService) ListIngresses(ctx context.Context, ns string, labelSelector map[string]string) (i *networkingv1beta1.IngressList, err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, ns, "ListIngresses", err == nil, t0)
	}(time.Now())
	return m.next.ListIngresses(ctx, ns, labelSelector)
}

// WatchIngresses satisfies controller.IngressControllerKubeService interface.
func (m MeasuredService) WatchIngresses(ctx context.Context, ns string, labelSelector map[string]string) (i watch.Interface, err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, ns, "WatchIngresses", err == nil, t0)
	}(time.Now())
	return m.next.WatchIngresses(ctx, ns, labelSelector)
}

// GetServiceHostAndPort satisifies security.KubeServiceTranslator interface.
func (m MeasuredService) GetServiceHostAndPort(ctx context.Context, svc model.KubernetesService) (h string, p int, err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveKubernetesServiceOperation(ctx, svc.Namespace, "GetServiceHostAndPort", err == nil, t0)
	}(time.Now())
	return m.next.GetServiceHostAndPort(ctx, svc)
}

var _ checkInterface = MeasuredService{}

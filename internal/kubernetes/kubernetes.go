package kubernetes

import (
	"context"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/slok/bilrost/internal/authbackend/dex"
	"github.com/slok/bilrost/internal/controller"
	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/model"
	"github.com/slok/bilrost/internal/proxy/oauth2proxy"
	"github.com/slok/bilrost/internal/security"
	authv1 "github.com/slok/bilrost/pkg/apis/auth/v1"
	kubernetesbilrost "github.com/slok/bilrost/pkg/kubernetes/gen/clientset/versioned"
)

// Service is the Kubernetes service that implements different interfaces around
// the app that are related with Kubernetes apiserver communication.
type Service struct {
	coreCli    kubernetes.Interface
	bilrostCli kubernetesbilrost.Interface
	logger     log.Logger
}

// NewService returns a new repository
func NewService(coreCli kubernetes.Interface, bilrostCli kubernetesbilrost.Interface, logger log.Logger) Service {
	return Service{
		bilrostCli: bilrostCli,
		coreCli:    coreCli,
		logger:     logger.WithKV(log.KV{"service": "kubernetes.Service"}),
	}
}

// GetAuthBackend satisfies multiple interfaces.
func (s Service) GetAuthBackend(_ context.Context, id string) (*model.AuthBackend, error) {
	logger := s.logger.WithKV(log.KV{"id": id})

	ab, err := s.bilrostCli.AuthV1().AuthBackends().Get(id, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	res := mapAuthBackendK8sToModel(ab)
	logger.Debugf("auth backends got")

	return res, nil
}

func mapAuthBackendK8sToModel(ab *authv1.AuthBackend) *model.AuthBackend {
	res := &model.AuthBackend{ID: ab.Name}

	switch {
	case ab.Spec.Dex != nil:
		res.Dex = &model.AuthBackendDex{
			APIURL:    ab.Spec.Dex.APIAddress,
			PublicURL: ab.Spec.Dex.PublicURL,
		}
	}

	return res
}

// EnsureDeployment satisfies multiple interfaces.
func (s Service) EnsureDeployment(_ context.Context, dep *appsv1.Deployment) error {
	logger := s.logger.WithKV(log.KV{"obj-ns": dep.Namespace, "obj-name": dep.Name})

	storedDep, err := s.coreCli.AppsV1().Deployments(dep.Namespace).Get(dep.Name, metav1.GetOptions{})
	if err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}
		_, err = s.coreCli.AppsV1().Deployments(dep.Namespace).Create(dep)
		if err != nil {
			return err
		}
		logger.Debugf("deployment has been created")

		return nil
	}

	// Force overwrite.
	dep.ObjectMeta.ResourceVersion = storedDep.ResourceVersion
	_, err = s.coreCli.AppsV1().Deployments(dep.Namespace).Update(dep)
	if err != nil {
		return err
	}
	logger.Debugf("deployment has been updated")

	return nil
}

// DeleteDeployment satisfies multiple interfaces.
func (s Service) DeleteDeployment(_ context.Context, ns, name string) error {
	logger := s.logger.WithKV(log.KV{"obj-ns": ns, "obj-name": name})

	err := s.coreCli.AppsV1().Deployments(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	logger.Debugf("deployment has been deleted")
	return nil
}

// ListDeployments satisfies multiple interfaces.
func (s Service) ListDeployments(_ context.Context, ns string, options metav1.ListOptions) (*appsv1.DeploymentList, error) {
	return s.coreCli.AppsV1().Deployments(ns).List(options)
}

// WatchDeployments satisfies multiple interfaces.
func (s Service) WatchDeployments(_ context.Context, ns string, options metav1.ListOptions) (watch.Interface, error) {
	return s.coreCli.AppsV1().Deployments(ns).Watch(options)
}

// EnsureService satisfies multiple interfaces.
func (s Service) EnsureService(_ context.Context, svc *corev1.Service) error {
	logger := s.logger.WithKV(log.KV{"obj-ns": svc.Namespace, "obj-name": svc.Name})

	storedSvc, err := s.coreCli.CoreV1().Services(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
	if err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}
		_, err = s.coreCli.CoreV1().Services(svc.Namespace).Create(svc)
		if err != nil {
			return err
		}
		logger.Debugf("service has been created")

		return nil
	}

	// Force overwrite.
	svc.ObjectMeta.ResourceVersion = storedSvc.ResourceVersion
	svc.Spec.ClusterIP = storedSvc.Spec.ClusterIP
	_, err = s.coreCli.CoreV1().Services(svc.Namespace).Update(svc)
	if err != nil {
		return err
	}
	logger.Debugf("service has been updated")

	return nil
}

// DeleteService satisfies multiple interfaces.
func (s Service) DeleteService(_ context.Context, ns, name string) error {
	logger := s.logger.WithKV(log.KV{"obj-ns": ns, "obj-name": name})

	err := s.coreCli.CoreV1().Services(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	logger.Debugf("service has been deleted")
	return nil
}

// ListServices satisfies multiple interfaces.
func (s Service) ListServices(_ context.Context, ns string, options metav1.ListOptions) (*corev1.ServiceList, error) {
	return s.coreCli.CoreV1().Services(ns).List(options)
}

// WatchServices satisfies multiple interfaces.
func (s Service) WatchServices(_ context.Context, ns string, options metav1.ListOptions) (watch.Interface, error) {
	return s.coreCli.CoreV1().Services(ns).Watch(options)
}

// GetSecret satisfies multiple interfaces.
func (s Service) GetSecret(_ context.Context, ns, name string) (*corev1.Secret, error) {
	logger := s.logger.WithKV(log.KV{"obj-ns": ns, "obj-name": name})

	secret, err := s.coreCli.CoreV1().Secrets(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	logger.Debugf("secret retrieved")

	return secret, nil
}

// EnsureSecret satisfies multiple interfaces.
func (s Service) EnsureSecret(_ context.Context, secret *corev1.Secret) error {
	logger := s.logger.WithKV(log.KV{"obj-ns": secret.Namespace, "obj-name": secret.Name})

	storedSecret, err := s.coreCli.CoreV1().Secrets(secret.Namespace).Get(secret.Name, metav1.GetOptions{})
	if err != nil {
		if !kubeerrors.IsNotFound(err) {
			return err
		}
		_, err = s.coreCli.CoreV1().Secrets(secret.Namespace).Create(secret)
		if err != nil {
			return err
		}
		logger.Debugf("secret has been created")

		return nil
	}

	// Force overwrite.
	secret.ObjectMeta.ResourceVersion = storedSecret.ResourceVersion
	_, err = s.coreCli.CoreV1().Secrets(secret.Namespace).Update(secret)
	if err != nil {
		return err
	}
	logger.Debugf("secret has been updated")

	return nil
}

// DeleteSecret satisfies multiple interfaces.
func (s Service) DeleteSecret(_ context.Context, ns, name string) error {
	logger := s.logger.WithKV(log.KV{"obj-ns": ns, "obj-name": name})

	err := s.coreCli.CoreV1().Secrets(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	logger.Debugf("secret has been deleted")
	return nil
}

// ListSecrets satisfies multiple interfaces.
func (s Service) ListSecrets(_ context.Context, ns string, options metav1.ListOptions) (*corev1.SecretList, error) {
	return s.coreCli.CoreV1().Secrets(ns).List(options)
}

// WatchSecrets satisfies multiple interfaces.
func (s Service) WatchSecrets(_ context.Context, ns string, options metav1.ListOptions) (watch.Interface, error) {
	return s.coreCli.CoreV1().Secrets(ns).Watch(options)
}

// GetIngress satisfies multiple interfaces.
func (s Service) GetIngress(_ context.Context, ns, name string) (*networkingv1beta1.Ingress, error) {
	logger := s.logger.WithKV(log.KV{"obj-ns": ns, "obj-name": name})

	ing, err := s.coreCli.NetworkingV1beta1().Ingresses(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	logger.Debugf("ingress got")

	return ing, nil
}

// UpdateIngress satisfies multiple interfaces.
func (s Service) UpdateIngress(_ context.Context, ingress *networkingv1beta1.Ingress) error {
	logger := s.logger.WithKV(log.KV{"obj-ns": ingress.Namespace, "obj-name": ingress.Name})

	_, err := s.coreCli.NetworkingV1beta1().Ingresses(ingress.Namespace).Update(ingress)
	if err != nil {
		return err
	}

	logger.Debugf("ingress updated")

	return nil
}

// ListIngresses satisfies multiple interfaces.
func (s Service) ListIngresses(_ context.Context, ns string, options metav1.ListOptions) (*networkingv1beta1.IngressList, error) {
	return s.coreCli.NetworkingV1beta1().Ingresses(ns).List(options)
}

// WatchIngresses satisfies multiple interfaces.
func (s Service) WatchIngresses(_ context.Context, ns string, options metav1.ListOptions) (watch.Interface, error) {
	return s.coreCli.NetworkingV1beta1().Ingresses(ns).Watch(options)
}

// GetServiceHostAndPort satisfies multiple interfaces.
func (s Service) GetServiceHostAndPort(_ context.Context, svc model.KubernetesService) (string, int, error) {
	host := fmt.Sprintf("%s.%s.svc.cluster.local", svc.Name, svc.Namespace)
	port, err := strconv.Atoi(svc.PortOrPortName)
	if err == nil {
		return host, port, nil
	}

	// Our port is based on a name.
	// TODO(slok): Should we optimize with DNS SRV resolution although is worse for development? make it optional?.
	service, err := s.coreCli.CoreV1().Services(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
	if err != nil {
		return "", 0, err
	}

	for _, port := range service.Spec.Ports {
		if port.Name == svc.PortOrPortName {
			return host, int(port.Port), nil
		}
	}

	return "", 0, fmt.Errorf("missing %s port name on service %s/%s", svc.PortOrPortName, svc.Namespace, svc.Name)
}

// checkInterface, is a custom internal type that has all the interfaces that our kubernetes.Service must satisfy
// we could do `var _ {MUST_IMPLEMENT_INTERFACE} = Service{}` for each of the interfaces, but this aggregated way
// we could do wrappers of `kubernetes.Service` that satisify this aggregated interface instead of declaring
// explicitly in all of them what are the interfaces that must implement.
// The use case can be seen on `metrics.go` in this same package.
// Requires Go 1.14 because of overlapped interfaces: https://github.com/golang/go/issues/6977
type checkInterface interface {
	security.AuthBackendRepository
	security.KubeServiceTranslator
	oauth2proxy.KubernetesRepository
	controller.HandlerKubernetesRepository
	controller.RetrieverKubernetesRepository
	dex.KubernetesRepository
}

var _ checkInterface = Service{}

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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

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

// GetAuthBackend satisifies controller.AuthBackendRepository interface.
func (s Service) GetAuthBackend(_ context.Context, id string) (*model.AuthBackend, error) {
	logger := s.logger.WithKV(log.KV{"id": id})

	ab, err := s.bilrostCli.AuthV1().AuthBackends().Get(id, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not retrieve auth backend from Kubernetes: %w", err)
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

// EnsureDeployment satisifes oauth2proxy.KubernetesRepository interface.
func (s Service) EnsureDeployment(_ context.Context, dep *appsv1.Deployment) error {
	logger := s.logger.WithKV(log.KV{"obj-ns": dep.Namespace, "obj-name": dep.Name})

	storedDep, err := s.coreCli.AppsV1().Deployments(dep.Namespace).Get(dep.Name, metav1.GetOptions{})
	if err != nil {
		if !kubeerrors.IsNotFound(err) {
			return fmt.Errorf("could not get deployment: %w", err)
		}
		_, err = s.coreCli.AppsV1().Deployments(dep.Namespace).Create(dep)
		if err != nil {
			return fmt.Errorf("could not create deployment: %w", err)
		}
		logger.Debugf("deployment has been created")

		return nil
	}

	// Force overwrite.
	dep.ObjectMeta.ResourceVersion = storedDep.ResourceVersion
	_, err = s.coreCli.AppsV1().Deployments(dep.Namespace).Update(dep)
	if err != nil {
		return fmt.Errorf("could not update deployment: %w", err)
	}
	logger.Debugf("deployment has been updated")

	return nil
}

// DeleteDeployment satisfies oauth2proxy.KubernetesRepository interface.
func (s Service) DeleteDeployment(_ context.Context, ns, name string) error {
	logger := s.logger.WithKV(log.KV{"obj-ns": ns, "obj-name": name})

	err := s.coreCli.AppsV1().Deployments(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("could not delete deployment: %w", err)
	}

	logger.Debugf("deployment has been deleted")
	return nil
}

// EnsureService satisifes oauth2proxy.KubernetesRepository interface.
func (s Service) EnsureService(_ context.Context, svc *corev1.Service) error {
	logger := s.logger.WithKV(log.KV{"obj-ns": svc.Namespace, "obj-name": svc.Name})

	storedSvc, err := s.coreCli.CoreV1().Services(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
	if err != nil {
		if !kubeerrors.IsNotFound(err) {
			return fmt.Errorf("could not get service: %w", err)
		}
		_, err = s.coreCli.CoreV1().Services(svc.Namespace).Create(svc)
		if err != nil {
			return fmt.Errorf("could not create service: %w", err)
		}
		logger.Debugf("service has been created")

		return nil
	}

	// Force overwrite.
	svc.ObjectMeta.ResourceVersion = storedSvc.ResourceVersion
	svc.Spec.ClusterIP = storedSvc.Spec.ClusterIP
	_, err = s.coreCli.CoreV1().Services(svc.Namespace).Update(svc)
	if err != nil {
		return fmt.Errorf("could not update service: %w", err)
	}
	logger.Debugf("service has been updated")

	return nil
}

// DeleteService satisfies oauth2proxy.KubernetesRepository interface.
func (s Service) DeleteService(_ context.Context, ns, name string) error {
	logger := s.logger.WithKV(log.KV{"obj-ns": ns, "obj-name": name})

	err := s.coreCli.CoreV1().Services(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("could not delete service: %w", err)
	}

	logger.Debugf("service has been deleted")
	return nil
}

// EnsureSecret satisifes oauth2proxy.KubernetesRepository interface.
func (s Service) EnsureSecret(_ context.Context, secret *corev1.Secret) error {
	logger := s.logger.WithKV(log.KV{"obj-ns": secret.Namespace, "obj-name": secret.Name})

	storedSecret, err := s.coreCli.CoreV1().Secrets(secret.Namespace).Get(secret.Name, metav1.GetOptions{})
	if err != nil {
		if !kubeerrors.IsNotFound(err) {
			return fmt.Errorf("could not get secret: %w", err)
		}
		_, err = s.coreCli.CoreV1().Secrets(secret.Namespace).Create(secret)
		if err != nil {
			return fmt.Errorf("could not create secret: %w", err)
		}
		logger.Debugf("secret has been created")

		return nil
	}

	// Force overwrite.
	secret.ObjectMeta.ResourceVersion = storedSecret.ResourceVersion
	_, err = s.coreCli.CoreV1().Secrets(secret.Namespace).Update(secret)
	if err != nil {
		return fmt.Errorf("could not update secrets: %w", err)
	}
	logger.Debugf("secret has been updated")

	return nil
}

// DeleteSecret satisfies oauth2proxy.KubernetesRepository interface.
func (s Service) DeleteSecret(_ context.Context, ns, name string) error {
	logger := s.logger.WithKV(log.KV{"obj-ns": ns, "obj-name": name})

	err := s.coreCli.CoreV1().Secrets(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("could not delete secret: %w", err)
	}

	logger.Debugf("secret has been deleted")
	return nil
}

// GetIngress satisfies oauth2proxy.KubernetesRepository interface.
func (s Service) GetIngress(ctx context.Context, ns, name string) (*networkingv1beta1.Ingress, error) {
	logger := s.logger.WithKV(log.KV{"obj-ns": ns, "obj-name": name})

	ing, err := s.coreCli.NetworkingV1beta1().Ingresses(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not retrieve ingress from Kubernetes: %w", err)
	}

	logger.Debugf("ingress got")

	return ing, nil
}

// UpdateIngress satisfies oauth2proxy.KubernetesRepository interface.
func (s Service) UpdateIngress(ctx context.Context, ingress *networkingv1beta1.Ingress) error {
	logger := s.logger.WithKV(log.KV{"obj-ns": ingress.Namespace, "obj-name": ingress.Name})

	_, err := s.coreCli.NetworkingV1beta1().Ingresses(ingress.Namespace).Update(ingress)
	if err != nil {
		return fmt.Errorf("could not update ingress in Kubernetes: %w", err)
	}

	logger.Debugf("ingress updated")

	return nil
}

// ListIngresses satisfies controller.IngressControllerKubeService interface.
func (s Service) ListIngresses(_ context.Context, ns string, labelSelector map[string]string) (*networkingv1beta1.IngressList, error) {
	return s.coreCli.NetworkingV1beta1().Ingresses(ns).List(metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector).String(),
	})
}

// WatchIngresses satisfies controller.IngressControllerKubeService interface.
func (s Service) WatchIngresses(ctx context.Context, ns string, labelSelector map[string]string) (watch.Interface, error) {
	return s.coreCli.NetworkingV1beta1().Ingresses(ns).Watch(metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector).String(),
	})
}

// GetServiceHostAndPort satisifies security.KubeServiceTranslator interface.
func (s Service) GetServiceHostAndPort(ctx context.Context, svc model.KubernetesService) (string, int, error) {
	host := fmt.Sprintf("%s.%s.svc.cluster.local", svc.Name, svc.Namespace)
	port, err := strconv.Atoi(svc.PortOrPortName)
	if err == nil {
		return host, port, nil
	}

	// Our port is based on a name.
	// TODO(slok): Should we optimize with DNS SRV resolution although is worse for development? make it optional?.
	service, err := s.coreCli.CoreV1().Services(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
	if err != nil {
		return "", 0, fmt.Errorf("could not get Kubernetes service %s/%s: %w", svc.Namespace, svc.Name, err)
	}

	for _, port := range service.Spec.Ports {
		if port.Name == svc.PortOrPortName {
			return host, int(port.Port), nil
		}
	}

	return "", 0, fmt.Errorf("missing %s port name on service %s/%s", svc.PortOrPortName, svc.Namespace, svc.Name)
}

// Interface implementation checks.
var (
	_ security.AuthBackendRepository   = Service{}
	_ security.KubeServiceTranslator   = Service{}
	_ oauth2proxy.KubernetesRepository = Service{}
	_ controller.KubernetesRepository  = Service{}
)

package oauth2proxy

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/proxy"
)

var (
	defaultScopes = []string{"openid", "email", "profile", "groups", "offline_access"}
)

// KubernetesRepository is the proxy kubernetes service used to communicate with Kubernetes.
type KubernetesRepository interface {
	EnsureDeployment(ctx context.Context, dep *appsv1.Deployment) error
	DeleteDeployment(ctx context.Context, ns, name string) error
	EnsureService(ctx context.Context, svc *corev1.Service) error
	DeleteService(ctx context.Context, ns, name string) error
	GetIngress(ctx context.Context, ns, name string) (*networkingv1beta1.Ingress, error)
	UpdateIngress(ctx context.Context, ingress *networkingv1beta1.Ingress) error
}

//go:generate mockery -case underscore -output oauth2proxymock -outpkg oauth2proxymock -name KubernetesRepository

type provisioner struct {
	kuberepo KubernetesRepository
	logger   log.Logger
}

// NewOIDCProvisioner returns a new oidc provisioner.
func NewOIDCProvisioner(kuberepo KubernetesRepository, logger log.Logger) proxy.OIDCProvisioner {
	return provisioner{
		kuberepo: kuberepo,
		logger:   logger.WithKV(log.KV{"service": "proxy.oauth2proxy.OIDCProvisioner"}),
	}
}

func (p provisioner) Provision(ctx context.Context, settings proxy.OIDCProxySettings) error {
	// Set defaults.
	if len(settings.Scopes) == 0 {
		settings.Scopes = defaultScopes
	}

	// Provision proxy.
	dep, err := p.provisionDeployment(ctx, settings)
	if err != nil {
		return fmt.Errorf("could not provision deployment on Kubernetes: %w", err)
	}

	err = p.provisionDeploymentService(ctx, dep)
	if err != nil {
		return fmt.Errorf("could not provision service on Kubernetes: %w", err)
	}

	// Point ingress to the secure proxy.
	err = p.setIngressToProxy(ctx, settings)
	if err != nil {
		return fmt.Errorf("could not update ingress in on Kubernetes to eanble oauth2 proxy service: %w", err)
	}

	return nil
}

func (p provisioner) provisionDeployment(ctx context.Context, settings proxy.OIDCProxySettings) (*appsv1.Deployment, error) {
	const proxyInternalPort = 4180
	name := getResourceName(settings.IngressName)
	replicas := int32(1)
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "bilrost",
		"app.kubernetes.io/name":       "oauth2-proxy",
		"app.kubernetes.io/component":  "proxy",
		"app.kubernetes.io/instance":   name,
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: settings.IngressNamespace,
			Labels:    labels,
			// TODO(slok): Use owner refs or apply our finalizers?.
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "quay.io/oauth2-proxy/oauth2-proxy:v5.1.0",
							Args: []string{
								fmt.Sprintf(`--oidc-issuer-url=%s`, settings.IssuerURL),
								fmt.Sprintf(`--client-id=%s`, settings.AppID),
								// TODO(slok): Create asecret and inject as env var.
								fmt.Sprintf(`--client-secret=%s`, settings.AppSecret),
								fmt.Sprintf(`--http-address=0.0.0.0:%d`, proxyInternalPort),
								fmt.Sprintf(`--redirect-url=%s/oauth2/callback`, settings.URL),
								fmt.Sprintf(`--upstream=%s`, settings.UpstreamURL),
								fmt.Sprintf(`--scope=%s`, strings.Join(settings.Scopes, " ")),
								`--cookie-secret=test`,
								`--cookie-secure=false`,
								`--provider=oidc`,
								`--skip-provider-button`,
								`--email-domain=*`,
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: proxyInternalPort,
									Name:          "http",
									Protocol:      "TCP",
								},
							},
							Resources: corev1.ResourceRequirements{
								// TODO(slok): Do we need limits?
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("15m"),
									corev1.ResourceMemory: resource.MustParse("20Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	err := p.kuberepo.EnsureDeployment(ctx, deployment)
	if err != nil {
		return nil, fmt.Errorf("could not set up proxy deployment: %w", err)
	}

	return deployment, nil
}

const (
	proxySvcPort = 80
	proxySvcName = "http"
)

func (p provisioner) provisionDeploymentService(ctx context.Context, dep *appsv1.Deployment) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dep.Name,
			Namespace: dep.Namespace,
			Labels:    dep.Labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     "ClusterIP",
			Selector: dep.Labels,
			Ports: []corev1.ServicePort{
				{
					Port:       proxySvcPort,
					Name:       proxySvcName,
					TargetPort: intstr.FromInt(int(dep.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)),
				},
			},
		},
	}

	err := p.kuberepo.EnsureService(ctx, svc)
	if err != nil {
		return fmt.Errorf("could not ensure proxy service: %w", err)
	}

	return nil
}

func (p provisioner) setIngressToProxy(ctx context.Context, settings proxy.OIDCProxySettings) error {
	proxyBackend := networkingv1beta1.IngressBackend{
		ServiceName: getResourceName(settings.IngressName),
		ServicePort: intstr.FromString(proxySvcName),
	}

	err := p.updateIngressBackend(ctx, settings.IngressNamespace, settings.IngressName, proxyBackend)
	if err != nil {
		return fmt.Errorf("could not point ingress to secured proxy: %w", err)
	}

	return nil
}

func (p provisioner) Unprovision(ctx context.Context, settings proxy.UnprovisionSettings) error {
	name := getResourceName(settings.IngressName)

	// Update ingress with original service.
	// Is important to make this as first step becase we don't want to be
	// unavailable if we delete the proxy.
	err := p.restoreIngress(ctx, settings)
	if err != nil {
		return fmt.Errorf("could not restore ingress previous value: %w", err)
	}

	// Delete Proxy.
	err = p.kuberepo.DeleteService(ctx, settings.IngressNamespace, name)
	if err != nil {
		return fmt.Errorf("could not unprovision proxy service: %w", err)
	}
	err = p.kuberepo.DeleteDeployment(ctx, settings.IngressNamespace, name)
	if err != nil {
		return fmt.Errorf("could not unprovision proxy deployment: %w", err)
	}

	return nil
}

func (p provisioner) restoreIngress(ctx context.Context, settings proxy.UnprovisionSettings) error {
	var port intstr.IntOrString
	if p, err := strconv.Atoi(settings.OriginalServicePortOrNamePort); err == nil {
		port = intstr.FromInt(p)
	} else {
		port = intstr.FromString(settings.OriginalServicePortOrNamePort)
	}

	origBackend := networkingv1beta1.IngressBackend{
		ServiceName: settings.OriginalServiceName,
		ServicePort: port,
	}

	err := p.updateIngressBackend(ctx, settings.IngressNamespace, settings.IngressName, origBackend)
	if err != nil {
		return fmt.Errorf("could not restore original ingress backend: %w", err)
	}

	return nil
}

func (p provisioner) updateIngressBackend(ctx context.Context, ns, name string, newBackend networkingv1beta1.IngressBackend) error {
	ing, err := p.kuberepo.GetIngress(ctx, ns, name)
	if err != nil {
		return err
	}

	// Pre checks of the ingress.
	rulesLen := len(ing.Spec.Rules)
	if rulesLen != 1 {
		return fmt.Errorf("ingress required rules is 1, got: %d", rulesLen)
	}
	pathsLen := len(ing.Spec.Rules[0].HTTP.Paths)
	if pathsLen != 1 {
		return fmt.Errorf("ingress required paths is 1, got: %d", pathsLen)
	}

	// Do we need to update the ingress?
	currentBackend := ing.Spec.Rules[0].HTTP.Paths[0].Backend
	if currentBackend == newBackend {
		p.logger.Debugf("ingress already pointing to %s:%s service, ignoring update", newBackend.ServiceName, newBackend.ServicePort)
		return nil
	}

	ing.Spec.Rules[0].HTTP.Paths[0].Backend = newBackend
	err = p.kuberepo.UpdateIngress(ctx, ing)
	if err != nil {
		return fmt.Errorf("could not update ingress with backend: %w", err)
	}

	return nil
}

func getResourceName(name string) string {
	return fmt.Sprintf("%s-bilrost-proxy", name)
}

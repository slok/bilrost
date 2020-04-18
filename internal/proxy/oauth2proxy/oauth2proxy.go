package oauth2proxy

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/proxy"
)

// KubernetesRepository is the proxy kubernetes service used to communicate with Kubernetes.
type KubernetesRepository interface {
	EnsureDeployment(ctx context.Context, dep *appsv1.Deployment) error
	EnsureService(ctx context.Context, svc *corev1.Service) error
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
	dep, err := p.provisionDeployment(ctx, settings)
	if err != nil {
		return fmt.Errorf("could not provision deployment on Kubernetes: %w", err)
	}

	err = p.provisionDeploymentService(ctx, dep)
	if err != nil {
		return fmt.Errorf("could not provision service on Kubernetes: %w", err)
	}

	return nil
}

func (p provisioner) provisionDeployment(ctx context.Context, settings proxy.OIDCProxySettings) (*appsv1.Deployment, error) {
	appNs, appName, err := getKubeNSAndNameFromID(settings.AppID)
	if err != nil {
		return nil, err
	}
	const proxyInternalPort = 4180
	name := fmt.Sprintf("%s-bilrost-proxy", appName)
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
			Namespace: appNs,
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
						corev1.Container{
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
								corev1.ContainerPort{
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

	err = p.kuberepo.EnsureDeployment(ctx, deployment)
	if err != nil {
		return nil, fmt.Errorf("could not set up proxy deployment: %w", err)
	}

	return deployment, nil
}

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
				corev1.ServicePort{
					Port:       80,
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

func (p provisioner) Unprovision(ctx context.Context, settings proxy.OIDCProxySettings) error {
	return nil
}

func getKubeNSAndNameFromID(id string) (ns, name string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("could not get ns and name from ID")
	}

	ns = parts[0]
	name = parts[1]
	if ns == "" {
		ns = "default"
	}

	return ns, name, nil
}

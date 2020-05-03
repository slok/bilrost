package oauth2proxy

import (
	"context"
	"crypto/md5"
	"encoding/json"
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

// KubernetesRepository is the proxy kubernetes service used to communicate with Kubernetes.
type KubernetesRepository interface {
	EnsureDeployment(ctx context.Context, dep *appsv1.Deployment) error
	DeleteDeployment(ctx context.Context, ns, name string) error
	EnsureService(ctx context.Context, svc *corev1.Service) error
	DeleteService(ctx context.Context, ns, name string) error
	EnsureSecret(ctx context.Context, sec *corev1.Secret) error
	DeleteSecret(ctx context.Context, ns, name string) error
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
	// Provision proxy.
	secret, err := p.provisionSecret(ctx, settings)
	if err != nil {
		return fmt.Errorf("could not provision secret on Kubernetes: %w", err)
	}

	dep, err := p.provisionDeployment(ctx, settings, secret)
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

const (
	oidcClientIDEnv     = "OIDC_CLIENT_ID"
	oidcClientSecretEnv = "OIDC_CLIENT_SECRET"
)

func (p provisioner) provisionSecret(ctx context.Context, settings proxy.OIDCProxySettings) (*corev1.Secret, error) {
	name := getResourceName(settings.App.Ingress.Name)
	labels := getLabels(name)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: settings.App.Ingress.Namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			oidcClientIDEnv:     []byte(settings.ClientID),
			oidcClientSecretEnv: []byte(settings.ClientSecret),
		},
	}

	err := p.kuberepo.EnsureSecret(ctx, secret)
	if err != nil {
		return nil, fmt.Errorf("could not ensure proxy secret: %w", err)
	}

	return secret, nil
}

func (p provisioner) provisionDeployment(ctx context.Context, settings proxy.OIDCProxySettings, secret *corev1.Secret) (*appsv1.Deployment, error) {
	const proxyInternalPort = 4180

	// For consistency we will create everything with the same names and labels.
	name := secret.Name
	ns := secret.Namespace
	labels := getLabels(name)

	// Small hack to automatically force a rolling deploy when the secrets change,
	// if we don't use something to force the rolling update the deployment will not
	// be updated although the secrets change.
	// More information: https://github.com/kubernetes/kubernetes/issues/22368
	checksumLabels := getLabels(name)
	checksum, err := secretChecksum(secret)
	if err != nil {
		return nil, fmt.Errorf("could not get checksum of secret data: %w", err)
	}
	checksumLabels["bilrost.slok.dev/secret-checksum-to-force-update"] = checksum

	customSettings := getCustomizableSettings(settings)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    labels,
			// TODO(slok): Use owner refs or apply our finalizers?.
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &customSettings.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: checksumLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: customSettings.Image,
							Args: []string{
								fmt.Sprintf(`--oidc-issuer-url=%s`, settings.IssuerURL),
								fmt.Sprintf(`--client-id=$(%s)`, oidcClientIDEnv),
								// TODO(slok): Create asecret and inject as env var.
								fmt.Sprintf(`--client-secret=$(%s)`, oidcClientSecretEnv),
								fmt.Sprintf(`--http-address=0.0.0.0:%d`, proxyInternalPort),
								fmt.Sprintf(`--redirect-url=%s/oauth2/callback`, settings.URL),
								fmt.Sprintf(`--upstream=%s`, settings.UpstreamURL),
								fmt.Sprintf(`--scope=%s`, strings.Join(customSettings.Scopes, " ")),
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
							Resources: customSettings.Resources,
							EnvFrom: []corev1.EnvFromSource{{
								SecretRef: &corev1.SecretEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: name,
									},
								},
							}},
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

type customizableSettings struct {
	Image     string
	Scopes    []string
	Replicas  int32
	Resources corev1.ResourceRequirements
}

func getCustomizableSettings(settings proxy.OIDCProxySettings) customizableSettings {
	defaults := customizableSettings{
		Image:    "quay.io/oauth2-proxy/oauth2-proxy:v5.1.0",
		Scopes:   []string{"openid", "email", "profile", "groups", "offline_access"},
		Replicas: int32(1),
		Resources: corev1.ResourceRequirements{
			// TODO(slok): Do we need limits?
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("15m"),
				corev1.ResourceMemory: resource.MustParse("20Mi"),
			},
		},
	}

	// Set custom settings.
	if len(settings.App.ProxySettings.Scopes) > 0 {
		defaults.Scopes = settings.App.ProxySettings.Scopes
	}

	if settings.App.ProxySettings.Oauth2Proxy == nil {
		return defaults
	}

	if settings.App.ProxySettings.Oauth2Proxy.Image != "" {
		defaults.Image = settings.App.ProxySettings.Oauth2Proxy.Image
	}
	if settings.App.ProxySettings.Oauth2Proxy.Replicas != 0 {
		defaults.Replicas = int32(settings.App.ProxySettings.Oauth2Proxy.Replicas)
	}

	if settings.App.ProxySettings.Oauth2Proxy.Resources != nil {
		defaults.Resources = *settings.App.ProxySettings.Oauth2Proxy.Resources
	}

	return defaults
}

const (
	proxySvcPort = 80
	proxySvcName = "http"
)

func (p provisioner) provisionDeploymentService(ctx context.Context, dep *appsv1.Deployment) error {
	// For consistency we will create everything with the same names and labels.
	name := dep.Name
	ns := dep.Namespace
	labels := getLabels(name)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     "ClusterIP",
			Selector: labels,
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
		ServiceName: getResourceName(settings.App.Ingress.Name),
		ServicePort: intstr.FromString(proxySvcName),
	}

	err := p.updateIngressBackend(ctx, settings.App.Ingress.Namespace, settings.App.Ingress.Name, proxyBackend)
	if err != nil {
		return fmt.Errorf("could not point ingress to secured proxy: %w", err)
	}

	return nil
}

func (p provisioner) Unprovision(ctx context.Context, settings proxy.UnprovisionSettings) error {
	name := getResourceName(settings.IngressName)
	ns := settings.IngressNamespace

	// Update ingress with original service.
	// Is important to make this as first step becase we don't want to be
	// unavailable if we delete the proxy.
	err := p.restoreIngress(ctx, settings)
	if err != nil {
		return fmt.Errorf("could not restore ingress previous value: %w", err)
	}

	// Delete Proxy.
	err = p.kuberepo.DeleteService(ctx, ns, name)
	if err != nil {
		return fmt.Errorf("could not unprovision proxy service: %w", err)
	}
	err = p.kuberepo.DeleteDeployment(ctx, ns, name)
	if err != nil {
		return fmt.Errorf("could not unprovision proxy deployment: %w", err)
	}
	err = p.kuberepo.DeleteSecret(ctx, ns, name)
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
		p.logger.Debugf("ingress already pointing to %s:%v service, ignoring update", newBackend.ServiceName)
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

func getLabels(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/managed-by": "bilrost",
		"app.kubernetes.io/name":       "oauth2-proxy",
		"app.kubernetes.io/component":  "proxy",
		"app.kubernetes.io/instance":   name,
	}
}

func secretChecksum(s *corev1.Secret) (string, error) {
	d, err := json.Marshal(s.Data)
	if err != nil {
		return "", err
	}

	checksum := md5.Sum([]byte(d))
	return fmt.Sprintf("%x", checksum), nil
}

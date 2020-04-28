package oauth2proxy_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/proxy"
	"github.com/slok/bilrost/internal/proxy/oauth2proxy"
	"github.com/slok/bilrost/internal/proxy/oauth2proxy/oauth2proxymock"
)

func getBaseSettings() proxy.OIDCProxySettings {
	return proxy.OIDCProxySettings{
		URL:              "https://my-app.my-cluster.dev",
		UpstreamURL:      "http://my-app.my-ns.svc.cluster.local:8080",
		IssuerURL:        "https://dex.my-cluster.dev",
		ClientID:         "my-app-bilrost",
		ClientSecret:     "my-secret",
		Scopes:           []string{"openid", "email", "profile", "groups", "offline_access"},
		IngressNamespace: "my-ns",
		IngressName:      "my-app",
	}
}

func getBaseLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/managed-by": "bilrost",
		"app.kubernetes.io/name":       "oauth2-proxy",
		"app.kubernetes.io/component":  "proxy",
		"app.kubernetes.io/instance":   "my-app-bilrost-proxy",
		"bilrost.slok.dev/src":         "dlsiqrjj5tmnibb1e1o0",
	}
}

func getBaseDeployment() *appsv1.Deployment {
	replicas := int32(1)
	checkSumLabels := getBaseLabels()
	checkSumLabels["bilrost.slok.dev/secret-checksum-to-force-update"] = "fcf3c8b4319dc73b54608b9fe857b39d"

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app-bilrost-proxy",
			Namespace: "my-ns",
			Labels:    getBaseLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: getBaseLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: checkSumLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "quay.io/oauth2-proxy/oauth2-proxy:v5.1.0",
							Args: []string{
								"--oidc-issuer-url=https://dex.my-cluster.dev",
								"--client-id=$(OIDC_CLIENT_ID)",
								"--client-secret=$(OIDC_CLIENT_SECRET)",
								"--http-address=0.0.0.0:4180",
								"--redirect-url=https://my-app.my-cluster.dev/oauth2/callback",
								"--upstream=http://my-app.my-ns.svc.cluster.local:8080",
								"--scope=openid email profile groups offline_access",
								`--cookie-secret=test`,
								`--cookie-secure=false`,
								`--provider=oidc`,
								"--skip-provider-button",
								`--email-domain=*`,
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 4180,
									Name:          "http",
									Protocol:      "TCP",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("15m"),
									corev1.ResourceMemory: resource.MustParse("20Mi"),
								},
							},
							EnvFrom: []corev1.EnvFromSource{{
								SecretRef: &corev1.SecretEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "my-app-bilrost-proxy",
									},
								},
							}},
						},
					},
				},
			},
		},
	}
}

func getBaseService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app-bilrost-proxy",
			Namespace: "my-ns",
			Labels:    getBaseLabels(),
		},
		Spec: corev1.ServiceSpec{
			Type:     "ClusterIP",
			Selector: getBaseLabels(),
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					Name:       "http",
					TargetPort: intstr.FromInt(4180),
				},
			},
		},
	}
}

func getBaseSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app-bilrost-proxy",
			Namespace: "my-ns",
			Labels:    getBaseLabels(),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"OIDC_CLIENT_ID":     []byte("my-app-bilrost"),
			"OIDC_CLIENT_SECRET": []byte("my-secret"),
		},
	}
}

func getBaseIngress() *networkingv1beta1.Ingress {
	return &networkingv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-app",
			Namespace:   "my-ns",
			Labels:      map[string]string{"test": "1"},
			Annotations: map[string]string{"test": "1"},
		},
		Spec: networkingv1beta1.IngressSpec{
			Rules: []networkingv1beta1.IngressRule{
				{
					IngressRuleValue: networkingv1beta1.IngressRuleValue{
						HTTP: &networkingv1beta1.HTTPIngressRuleValue{
							Paths: []networkingv1beta1.HTTPIngressPath{
								{
									Backend: networkingv1beta1.IngressBackend{
										ServiceName: "my-app",
										ServicePort: intstr.FromInt(8080),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestOIDCProvisionerProvision(t *testing.T) {
	tests := map[string]struct {
		settings func() proxy.OIDCProxySettings
		mock     func(m *oauth2proxymock.KubernetesRepository)
		expErr   bool
	}{
		"A correct proxy provisioning should provision a secret, a deployment, a service, and swap the ingress.": {
			settings: getBaseSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				expSec := getBaseSecret()
				expDep := getBaseDeployment()
				expSvc := getBaseService()

				m.On("EnsureSecret", mock.Anything, expSec).Once().Return(nil)
				m.On("EnsureDeployment", mock.Anything, expDep).Once().Return(nil)
				m.On("EnsureService", mock.Anything, expSvc).Once().Return(nil)

				storedIngress := getBaseIngress()
				m.On("GetIngress", mock.Anything, "my-ns", "my-app").Once().Return(storedIngress, nil)

				expIngress := getBaseIngress()
				expIngress.Spec.Rules[0].HTTP.Paths[0].Backend = networkingv1beta1.IngressBackend{
					ServiceName: "my-app-bilrost-proxy",
					ServicePort: intstr.FromString("http"),
				}
				m.On("UpdateIngress", mock.Anything, expIngress).Once().Return(nil)
			},
		},

		"If stored ingress already has been swapped, it shouldn't be updated.": {
			settings: getBaseSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				expSec := getBaseSecret()
				expDep := getBaseDeployment()
				expSvc := getBaseService()

				m.On("EnsureSecret", mock.Anything, expSec).Once().Return(nil)
				m.On("EnsureDeployment", mock.Anything, expDep).Once().Return(nil)
				m.On("EnsureService", mock.Anything, expSvc).Once().Return(nil)

				storedIngress := getBaseIngress()
				storedIngress.Spec.Rules[0].HTTP.Paths[0].Backend = networkingv1beta1.IngressBackend{
					ServiceName: "my-app-bilrost-proxy",
					ServicePort: intstr.FromString("http"),
				}
				m.On("GetIngress", mock.Anything, "my-ns", "my-app").Once().Return(storedIngress, nil)
			},
		},

		"Failing setting up the secret should stop the provision process.": {
			settings: getBaseSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				m.On("EnsureSecret", mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing setting up the deployment should stop the provision process.": {
			settings: getBaseSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				m.On("EnsureSecret", mock.Anything, mock.Anything).Once().Return(nil)
				m.On("EnsureDeployment", mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing setting up the service should stop the provision process.": {
			settings: getBaseSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				m.On("EnsureSecret", mock.Anything, mock.Anything).Once().Return(nil)
				m.On("EnsureDeployment", mock.Anything, mock.Anything).Once().Return(nil)
				m.On("EnsureService", mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing getting the original ingress should stop the provision process.": {
			settings: getBaseSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				m.On("EnsureSecret", mock.Anything, mock.Anything).Once().Return(nil)
				m.On("EnsureDeployment", mock.Anything, mock.Anything).Once().Return(nil)
				m.On("EnsureService", mock.Anything, mock.Anything).Once().Return(nil)
				m.On("GetIngress", mock.Anything, mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing updating app ingress should stop the provision process.": {
			settings: getBaseSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				m.On("EnsureSecret", mock.Anything, mock.Anything).Once().Return(nil)
				m.On("EnsureDeployment", mock.Anything, mock.Anything).Once().Return(nil)
				m.On("EnsureService", mock.Anything, mock.Anything).Once().Return(nil)
				m.On("GetIngress", mock.Anything, mock.Anything, mock.Anything).Once().Return(getBaseIngress(), nil)
				m.On("UpdateIngress", mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			m := &oauth2proxymock.KubernetesRepository{}
			test.mock(m)

			prov := oauth2proxy.NewOIDCProvisioner(m, log.Dummy)
			err := prov.Provision(context.TODO(), test.settings())

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				m.AssertExpectations(t)
			}
		})
	}
}

func getBaseUnprovisionSettings() proxy.UnprovisionSettings {
	return proxy.UnprovisionSettings{
		IngressName:                   "test",
		IngressNamespace:              "test-ns",
		OriginalServiceName:           "test-orig-svc",
		OriginalServicePortOrNamePort: "http-orig",
	}
}

func TestOIDCProvisionerUnprovision(t *testing.T) {
	tests := map[string]struct {
		settings func() proxy.UnprovisionSettings
		mock     func(m *oauth2proxymock.KubernetesRepository)
		expErr   bool
	}{
		"A correct proxy unprovisioning should restore the original ingress and GC the proxy.": {
			settings: getBaseUnprovisionSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				storedIng := getBaseIngress()
				m.On("GetIngress", context.TODO(), "test-ns", "test").Once().Return(storedIng, nil)

				expIngress := storedIng.DeepCopy()
				expIngress.Spec.Rules[0].HTTP.Paths[0].Backend = networkingv1beta1.IngressBackend{
					ServiceName: "test-orig-svc",
					ServicePort: intstr.FromString("http-orig"),
				}
				m.On("UpdateIngress", context.TODO(), expIngress).Once().Return(nil)
				m.On("DeleteService", context.TODO(), "test-ns", "test-bilrost-proxy").Once().Return(nil)
				m.On("DeleteDeployment", context.TODO(), "test-ns", "test-bilrost-proxy").Once().Return(nil)
				m.On("DeleteSecret", context.TODO(), "test-ns", "test-bilrost-proxy").Once().Return(nil)
			},
		},

		"If stored ingress already has been restored, it shouldn't be updated.": {
			settings: getBaseUnprovisionSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				storedIng := getBaseIngress()
				storedIng.Spec.Rules[0].HTTP.Paths[0].Backend = networkingv1beta1.IngressBackend{
					ServiceName: "test-orig-svc",
					ServicePort: intstr.FromString("http-orig"),
				}
				m.On("GetIngress", context.TODO(), "test-ns", "test").Once().Return(storedIng, nil)
				m.On("DeleteService", context.TODO(), "test-ns", "test-bilrost-proxy").Once().Return(nil)
				m.On("DeleteDeployment", context.TODO(), "test-ns", "test-bilrost-proxy").Once().Return(nil)
				m.On("DeleteSecret", context.TODO(), "test-ns", "test-bilrost-proxy").Once().Return(nil)
			},
		},

		"Failing getting the ingress should stop the process.": {
			settings: getBaseUnprovisionSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				m.On("GetIngress", context.TODO(), mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing restoring the ingress should stop the process.": {
			settings: getBaseUnprovisionSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				m.On("GetIngress", context.TODO(), mock.Anything, mock.Anything).Once().Return(getBaseIngress(), nil)
				m.On("UpdateIngress", context.TODO(), mock.Anything).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing deleting the proxy service should stop the process.": {
			settings: getBaseUnprovisionSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				m.On("GetIngress", context.TODO(), mock.Anything, mock.Anything).Once().Return(getBaseIngress(), nil)
				m.On("UpdateIngress", context.TODO(), mock.Anything).Once().Return(nil)
				m.On("DeleteService", context.TODO(), mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing deleting the proxy deployment should stop the process.": {
			settings: getBaseUnprovisionSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				m.On("GetIngress", context.TODO(), mock.Anything, mock.Anything).Once().Return(getBaseIngress(), nil)
				m.On("UpdateIngress", context.TODO(), mock.Anything).Once().Return(nil)
				m.On("DeleteService", context.TODO(), mock.Anything, mock.Anything).Once().Return(nil)
				m.On("DeleteDeployment", context.TODO(), mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing deleting the proxy secret should stop the process.": {
			settings: getBaseUnprovisionSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				m.On("GetIngress", context.TODO(), mock.Anything, mock.Anything).Once().Return(getBaseIngress(), nil)
				m.On("UpdateIngress", context.TODO(), mock.Anything).Once().Return(nil)
				m.On("DeleteService", context.TODO(), mock.Anything, mock.Anything).Once().Return(nil)
				m.On("DeleteDeployment", context.TODO(), mock.Anything, mock.Anything).Once().Return(nil)
				m.On("DeleteSecret", context.TODO(), mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			m := &oauth2proxymock.KubernetesRepository{}
			test.mock(m)

			prov := oauth2proxy.NewOIDCProvisioner(m, log.Dummy)
			err := prov.Unprovision(context.TODO(), test.settings())

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				m.AssertExpectations(t)
			}
		})
	}
}

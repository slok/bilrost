package oauth2proxy_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/proxy"
	"github.com/slok/bilrost/internal/proxy/oauth2proxy"
	"github.com/slok/bilrost/internal/proxy/oauth2proxy/oauth2proxymock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func getBaseSettings() proxy.OIDCProxySettings {
	return proxy.OIDCProxySettings{
		URL:         "https://my-app.my-cluster.dev",
		UpstreamURL: "http://my-app.my-ns.svc.cluster.local:8080",
		IssuerURL:   "https://dex.my-cluster.dev",
		AppID:       "my-ns/my-app",
		AppSecret:   "my-secret",
		Scopes:      []string{"openid", "email", "profile", "groups", "offline_access"},
	}
}

var baseLabels = map[string]string{
	"app.kubernetes.io/managed-by": "bilrost",
	"app.kubernetes.io/name":       "oauth2-proxy",
	"app.kubernetes.io/component":  "proxy",
	"app.kubernetes.io/instance":   "my-app-bilrost-proxy",
}

func getBaseDeployment() *appsv1.Deployment {
	replicas := int32(1)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app-bilrost-proxy",
			Namespace: "my-ns",
			Labels:    baseLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: baseLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: baseLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name:  "app",
							Image: "quay.io/oauth2-proxy/oauth2-proxy:v5.1.0",
							Args: []string{
								"--oidc-issuer-url=https://dex.my-cluster.dev",
								"--client-id=my-ns/my-app",
								"--client-secret=my-secret",
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
								corev1.ContainerPort{
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
			Labels:    baseLabels,
		},
		Spec: corev1.ServiceSpec{
			Type:     "ClusterIP",
			Selector: baseLabels,
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Port:       80,
					TargetPort: intstr.FromInt(4180),
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
		"A correct proxy provisioning should provision a deployment and a service.": {
			settings: getBaseSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				expDep := getBaseDeployment()
				expSvc := getBaseService()

				m.On("EnsureDeployment", mock.Anything, expDep).Once().Return(nil)
				m.On("EnsureService", mock.Anything, expSvc).Once().Return(nil)
			},
		},

		"A wrong app ID without namespace should stop the process.": {
			settings: func() proxy.OIDCProxySettings {
				s := getBaseSettings()
				s.AppID = "test"
				return s
			},
			mock:   func(m *oauth2proxymock.KubernetesRepository) {},
			expErr: true,
		},

		"Failing setting up the deployment should stop the provision process.": {
			settings: getBaseSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				m.On("EnsureDeployment", mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing setting up the service should stop the provision process.": {
			settings: getBaseSettings,
			mock: func(m *oauth2proxymock.KubernetesRepository) {
				m.On("EnsureDeployment", mock.Anything, mock.Anything).Once().Return(nil)
				m.On("EnsureService", mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
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

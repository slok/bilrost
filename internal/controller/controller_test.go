package controller_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/slok/bilrost/internal/controller"
	"github.com/slok/bilrost/internal/controller/controllermock"
	"github.com/slok/bilrost/internal/model"
	"github.com/slok/bilrost/internal/security/securitymock"
	authv1 "github.com/slok/bilrost/pkg/apis/auth/v1"
)

func getBaseIngress() *networkingv1beta1.Ingress {
	return &networkingv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
			Labels:    map[string]string{"in-test": "true"},
		},
		Spec: networkingv1beta1.IngressSpec{
			Rules: []networkingv1beta1.IngressRule{
				{
					Host: "https://bilrost-controller-test.slok.dev",
					IngressRuleValue: networkingv1beta1.IngressRuleValue{
						HTTP: &networkingv1beta1.HTTPIngressRuleValue{
							Paths: []networkingv1beta1.HTTPIngressPath{
								{
									Backend: networkingv1beta1.IngressBackend{
										ServiceName: "my-app",
										ServicePort: intstr.FromString("http"),
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

func getBaseIngressAuth() *authv1.IngressAuth {
	return &authv1.IngressAuth{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
			Labels:    map[string]string{"in-test": "true"},
		},
		Spec: authv1.IngressAuthSpec{
			AuthSettings: authv1.AuthSettings{
				ScopeOrClaims: []string{"c1", "c2", "c3"},
			},
			AuthProxySource: authv1.AuthProxySource{
				Oauth2Proxy: &authv1.Oauth2ProxyAuthProxySource{
					Image:    "quay.io/oauth2-proxy/oauth2-proxy:v5.1.0",
					Replicas: 4,
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("15m"),
							corev1.ResourceMemory: resource.MustParse("20Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("30m"),
							corev1.ResourceMemory: resource.MustParse("45Mi"),
						},
					},
				},
			},
		},
	}
}

func getBaseApp() model.App {
	return model.App{
		ID:            "test-ns/test",
		AuthBackendID: "test-backend-id",
		Host:          "https://bilrost-controller-test.slok.dev",
		Ingress: model.KubernetesIngress{
			Name:      "test",
			Namespace: "test-ns",
			Upstream: model.KubernetesService{
				Name:           "my-app",
				Namespace:      "test-ns",
				PortOrPortName: "http",
			},
		},
	}
}

func getAdvancedApp() model.App {
	return model.App{
		ID:            "test-ns/test",
		AuthBackendID: "test-backend-id",
		Host:          "https://bilrost-controller-test.slok.dev",
		Ingress: model.KubernetesIngress{
			Name:      "test",
			Namespace: "test-ns",
			Upstream: model.KubernetesService{
				Name:           "my-app",
				Namespace:      "test-ns",
				PortOrPortName: "http",
			},
		},
		ProxySettings: model.ProxySettings{
			Scopes: []string{"c1", "c2", "c3"},
			Oauth2Proxy: &model.Oauth2ProxySettings{
				Image:    "quay.io/oauth2-proxy/oauth2-proxy:v5.1.0",
				Replicas: 4,
				Resources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("15m"),
						corev1.ResourceMemory: resource.MustParse("20Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("30m"),
						corev1.ResourceMemory: resource.MustParse("45Mi"),
					},
				},
			},
		},
	}
}

func TestHandler(t *testing.T) {
	tests := map[string]struct {
		obj    func() runtime.Object
		mock   func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service)
		expErr bool
	}{
		"If we try handling an object that we are not supose to handle it should not be handled.": {
			obj: func() runtime.Object {
				return &corev1.Pod{}
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {},
		},

		"An IngressAuth object should try getting the ingress and if errors, fail the handling.": {
			obj: func() runtime.Object {
				return getBaseIngressAuth()
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {
				mkr.On("GetIngress", mock.Anything, mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"An ingressAuth should get the ingress and start a regular ingress handling (without controller annotations should be ignored).": {
			obj: func() runtime.Object {
				return getBaseIngressAuth()
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {
				ing := getBaseIngress()
				ing.Annotations = map[string]string{}
				mkr.On("GetIngress", mock.Anything, "test-ns", "test").Once().Return(ing, nil)
			},
		},

		"An ingress without controller annotations should be ignored.": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Annotations = map[string]string{}
				return ing
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {},
		},

		"An ingress with empty/blank backend controller annotation should be ignored.": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "",
				}
				return ing
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {},
		},

		"An ingress with more than 1 ingress rule should error.": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
				}
				ing.Spec.Rules = append(ing.Spec.Rules, networkingv1beta1.IngressRule{})
				return ing
			},
			mock:   func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {},
			expErr: true,
		},

		"An ingress with more than 1 HTTP paths should error.": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
				}
				ing.Spec.Rules[0].HTTP.Paths = append(ing.Spec.Rules[0].HTTP.Paths, networkingv1beta1.HTTPIngressPath{})
				return ing
			},
			mock:   func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {},
			expErr: true,
		},

		"An ingress that is not ready but should be handled should be set ready to be handled on next iterations.": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
				}
				return ing
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
				}
				ing.Finalizers = []string{
					"test1",
					"test2",
				}
				mkr.On("GetIngress", mock.Anything, "test-ns", "test").Once().Return(ing, nil)

				// Marked as handled and with finalizer.
				expIng := getBaseIngress()
				expIng.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
					"auth.bilrost.slok.dev/handled": "true",
				}
				expIng.Finalizers = []string{
					"test1",
					"test2",
					"finalizers.auth.bilrost.slok.dev/security",
				}
				mkr.On("UpdateIngress", mock.Anything, expIng).Once().Return(nil)
			},
		},

		"An ingress that is ready to be handled should be secured (internal ready marks not mutated by 3rd parties).": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
					"auth.bilrost.slok.dev/handled": "true",
				}
				return ing
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {
				mkr.On("GetIngressAuth", mock.Anything, mock.Anything, mock.Anything).Once().Return(&authv1.IngressAuth{}, nil)

				// Secure process.
				ms.On("SecureApp", mock.Anything, mock.Anything).Once().Return(nil)

				// Our ingress is ok.
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
					"auth.bilrost.slok.dev/handled": "true",
				}
				ing.Finalizers = []string{"finalizers.auth.bilrost.slok.dev/security"}
				mkr.On("GetIngress", mock.Anything, "test-ns", "test").Once().Return(ing, nil)
			},
		},

		"An ingress that is ready to be handled should be secured (internal ready marks mutated by 3rd parties, requires healing ready marks).": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
					"auth.bilrost.slok.dev/handled": "true",
				}
				return ing
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {
				mkr.On("GetIngressAuth", mock.Anything, mock.Anything, mock.Anything).Once().Return(&authv1.IngressAuth{}, nil)

				// Secure process.
				ms.On("SecureApp", mock.Anything, mock.Anything).Once().Return(nil)

				// Some user or controller has deleted our marks.
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
				}
				mkr.On("GetIngress", mock.Anything, "test-ns", "test").Once().Return(ing, nil)

				// Marked as handled and with finalizer.
				expIng := getBaseIngress()
				expIng.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
					"auth.bilrost.slok.dev/handled": "true",
				}
				expIng.Finalizers = []string{"finalizers.auth.bilrost.slok.dev/security"}
				mkr.On("UpdateIngress", mock.Anything, expIng).Once().Return(nil)
			},
		},

		"An ingress that is ready to be handled should be secured (with advanced options from IngressAuth CR).": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
					"auth.bilrost.slok.dev/handled": "true",
				}
				return ing
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {
				ia := getBaseIngressAuth()
				mkr.On("GetIngressAuth", mock.Anything, "test-ns", "test").Once().Return(ia, nil)

				// Secure process with advanced options (check mapping correct).
				expApp := getAdvancedApp()
				ms.On("SecureApp", mock.Anything, expApp).Once().Return(nil)

				// Some user or controller has deleted our marks.
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
				}
				mkr.On("GetIngress", mock.Anything, "test-ns", "test").Once().Return(ing, nil)

				// Marked as handled and with finalizer.
				expIng := getBaseIngress()
				expIng.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
					"auth.bilrost.slok.dev/handled": "true",
				}
				expIng.Finalizers = []string{"finalizers.auth.bilrost.slok.dev/security"}
				mkr.On("UpdateIngress", mock.Anything, expIng).Once().Return(nil)
			},
		},

		"An ingress that was already handled without backend annotation should rollback and unmark.": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/handled": "true",
				}
				return ing
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {
				mkr.On("GetIngressAuth", mock.Anything, mock.Anything, mock.Anything).Once().Return(&authv1.IngressAuth{}, nil)

				// Rollback process.
				expApp := getBaseApp()
				expApp.AuthBackendID = "" // Because we don't have this.
				ms.On("RollbackAppSecurity", mock.Anything, expApp).Once().Return(nil)

				// Unmark as handled and remove the finalizer.
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/handled": "true",
				}
				ing.Finalizers = []string{
					"test1",
					"finalizers.auth.bilrost.slok.dev/security",
					"test2",
				}
				mkr.On("GetIngress", mock.Anything, "test-ns", "test").Once().Return(ing, nil)

				expIng := ing.DeepCopy()
				expIng.Annotations = map[string]string{}
				expIng.Finalizers = []string{
					"test1",
					"test2",
				}
				mkr.On("UpdateIngress", mock.Anything, expIng).Once().Return(nil)
			},
		},

		"An ingress that has been deleted should be rollback and unmark.": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
					"auth.bilrost.slok.dev/handled": "true",
				}
				ing.Finalizers = []string{"finalizers.auth.bilrost.slok.dev/security"}
				ing.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				return ing
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {
				mkr.On("GetIngressAuth", mock.Anything, mock.Anything, mock.Anything).Once().Return(&authv1.IngressAuth{}, nil)

				// Rollback process.
				expApp := getBaseApp()
				ms.On("RollbackAppSecurity", mock.Anything, expApp).Once().Return(nil)

				// Unmark as handled and remove the finalizer.
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
					"auth.bilrost.slok.dev/handled": "true",
				}
				ing.Finalizers = []string{"finalizers.auth.bilrost.slok.dev/security"}
				mkr.On("GetIngress", mock.Anything, "test-ns", "test").Once().Return(ing, nil)

				expIng := ing.DeepCopy()
				expIng.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
				}
				expIng.Finalizers = []string{}
				mkr.On("UpdateIngress", mock.Anything, expIng).Once().Return(nil)
			},
		},

		"An ingress that has been deleted should be rollback and unmark (with advanced options from IngressAuth CR).": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
					"auth.bilrost.slok.dev/handled": "true",
				}
				ing.Finalizers = []string{"finalizers.auth.bilrost.slok.dev/security"}
				ing.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				return ing
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {
				ia := getBaseIngressAuth()
				mkr.On("GetIngressAuth", mock.Anything, mock.Anything, mock.Anything).Once().Return(ia, nil)

				// Rollback process.
				expApp := getAdvancedApp()
				ms.On("RollbackAppSecurity", mock.Anything, expApp).Once().Return(nil)

				// Unmark as handled and remove the finalizer.
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
					"auth.bilrost.slok.dev/handled": "true",
				}
				ing.Finalizers = []string{"finalizers.auth.bilrost.slok.dev/security"}
				mkr.On("GetIngress", mock.Anything, "test-ns", "test").Once().Return(ing, nil)

				expIng := ing.DeepCopy()
				expIng.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
				}
				expIng.Finalizers = []string{}
				mkr.On("UpdateIngress", mock.Anything, expIng).Once().Return(nil)
			},
		},

		"An ingress that has been deleted and already cleaned shoudl be ignored.": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/backend": "test-backend-id",
					"auth.bilrost.slok.dev/handled": "true",
				}
				ing.Finalizers = []string{}
				ing.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				return ing
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Mocks.
			mkr := &controllermock.HandlerKubernetesRepository{}
			ms := &securitymock.Service{}
			test.mock(mkr, ms)

			// Run.
			cfg := controller.HandlerConfig{
				KubernetesRepo: mkr,
				SecuritySvc:    ms,
			}
			h, err := controller.NewHandler(cfg)
			require.NoError(err)
			err = h.Handle(context.TODO(), test.obj())

			// Check.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				mkr.AssertExpectations(t)
				ms.AssertExpectations(t)
			}
		})
	}
}

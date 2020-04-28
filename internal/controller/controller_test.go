package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/slok/bilrost/internal/controller"
	"github.com/slok/bilrost/internal/controller/controllermock"
	"github.com/slok/bilrost/internal/model"
	"github.com/slok/bilrost/internal/security/securitymock"
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

func getBaseSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "whatever",
			Namespace: "whatever-ns",
			Labels: map[string]string{
				"bilrost.slok.dev/src": "ehin6t1ddppiut35edq2qrj1dlig",
			},
		},
	}
}

func getBaseService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "whatever",
			Namespace: "whatever-ns",
			Labels: map[string]string{
				"bilrost.slok.dev/src": "ehin6t1ddppiut35edq2qrj1dlig",
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

		"The controller should react on tracked secret events (without ingress annotation).": {
			obj: func() runtime.Object {
				s := getBaseSecret()
				s.Labels = map[string]string{}
				return s
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {},
		},

		"The controller should react on tracked secret events (with ingress annotation).": {
			obj: func() runtime.Object {
				return getBaseSecret()

			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {
				expIng := getBaseIngress()
				mkr.On("GetIngress", mock.Anything, "test-ns", "test-name").Once().Return(expIng, nil)
			},
		},

		"The controller should react on tracked service events (without ingress annotation).": {
			obj: func() runtime.Object {
				s := getBaseService()
				s.Labels = map[string]string{}
				return s
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {},
		},

		"The controller should react on tracked service events (with ingress annotation).": {
			obj: func() runtime.Object {
				return getBaseService()
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {
				expIng := getBaseIngress()
				mkr.On("GetIngress", mock.Anything, "test-ns", "test-name").Once().Return(expIng, nil)
			},
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

		"An ingress that was already handled without backend annotation should rollback and unmark.": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Annotations = map[string]string{
					"auth.bilrost.slok.dev/handled": "true",
				}
				return ing
			},
			mock: func(mkr *controllermock.HandlerKubernetesRepository, ms *securitymock.Service) {
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

		"An ingress that has been deleted  and already cleaned shoudl be ignored.": {
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

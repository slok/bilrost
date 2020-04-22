package controller_test

import (
	"context"
	"testing"

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
			Annotations: map[string]string{
				"auth.bilrost.slok.dev/backend": "test-backend-id",
			},
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

func TestHandler(t *testing.T) {
	tests := map[string]struct {
		obj    func() runtime.Object
		mock   func(mkr *controllermock.KubernetesRepository, ms *securitymock.Service)
		expErr bool
	}{
		"If we try handling an object that we are not supose to handle it should not handle.": {
			obj: func() runtime.Object {
				return &corev1.Pod{}
			},
			mock: func(mkr *controllermock.KubernetesRepository, ms *securitymock.Service) {},
		},

		"An ingress without controller annotations should be ignored.": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				delete(ing.Annotations, "auth.bilrost.slok.dev/backend")
				return ing
			},
			mock: func(mkr *controllermock.KubernetesRepository, ms *securitymock.Service) {},
		},

		"An ingress with empty controller annotations should be ignored.": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Annotations["auth.bilrost.slok.dev/backend"] = ""
				return ing
			},
			mock: func(mkr *controllermock.KubernetesRepository, ms *securitymock.Service) {},
		},

		"An ingress with more than 1 ingress rule should error.": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Spec.Rules = append(ing.Spec.Rules, networkingv1beta1.IngressRule{})
				return ing
			},
			mock:   func(mkr *controllermock.KubernetesRepository, ms *securitymock.Service) {},
			expErr: true,
		},

		"An ingress with more than 1 HTTP paths should error.": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Spec.Rules[0].HTTP.Paths = append(ing.Spec.Rules[0].HTTP.Paths, networkingv1beta1.HTTPIngressPath{})
				return ing
			},
			mock:   func(mkr *controllermock.KubernetesRepository, ms *securitymock.Service) {},
			expErr: true,
		},

		"An ingress that wasn't handled before with backend annotation should be secured and marked as hanlded.": {
			obj: func() runtime.Object {
				return getBaseIngress()
			},
			mock: func(mkr *controllermock.KubernetesRepository, ms *securitymock.Service) {
				// Secure process.
				expApp := getBaseApp()
				ms.On("SecureApp", mock.Anything, expApp).Once().Return(nil)

				// Mark as handled process.
				ing := getBaseIngress()
				mkr.On("GetIngress", mock.Anything, "test-ns", "test").Once().Return(ing, nil)
				handledIng := ing.DeepCopy()
				handledIng.Annotations["auth.bilrost.slok.dev/handled"] = "true"
				mkr.On("UpdateIngress", mock.Anything, handledIng).Once().Return(nil)
			},
		},

		"An ingress that was already handled with backend annotation should be secured and not marked as hanlded.": {
			obj: func() runtime.Object {
				return getBaseIngress()
			},
			mock: func(mkr *controllermock.KubernetesRepository, ms *securitymock.Service) {
				// Secure process.
				ms.On("SecureApp", mock.Anything, mock.Anything).Once().Return(nil)

				// Mark as handled.
				ing := getBaseIngress()
				ing.Annotations["auth.bilrost.slok.dev/handled"] = "true"
				mkr.On("GetIngress", mock.Anything, "test-ns", "test").Once().Return(ing, nil)
			},
		},

		"An ingress that was already handled without backend annotation should rollback and unmark.": {
			obj: func() runtime.Object {
				ing := getBaseIngress()
				ing.Annotations["auth.bilrost.slok.dev/handled"] = "true"
				delete(ing.Annotations, "auth.bilrost.slok.dev/backend")
				return ing
			},
			mock: func(mkr *controllermock.KubernetesRepository, ms *securitymock.Service) {
				// Rollback process.
				expApp := getBaseApp()
				expApp.AuthBackendID = "" // Because we don't have this.
				ms.On("RollbackAppSecurity", mock.Anything, expApp).Once().Return(nil)

				// Unmark as handled.
				ing := getBaseIngress()
				ing.Annotations["auth.bilrost.slok.dev/handled"] = "true"
				mkr.On("GetIngress", mock.Anything, "test-ns", "test").Once().Return(ing, nil)
				handledIng := ing.DeepCopy()
				delete(handledIng.Annotations, "auth.bilrost.slok.dev/handled")
				mkr.On("UpdateIngress", mock.Anything, handledIng).Once().Return(nil)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Mocks.
			mkr := &controllermock.KubernetesRepository{}
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

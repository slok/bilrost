package controller

import (
	"fmt"

	networkingv1beta1 "k8s.io/api/networking/v1beta1"

	"github.com/slok/bilrost/internal/model"
	authv1 "github.com/slok/bilrost/pkg/apis/auth/v1"
)

// maps an ingress and a ingress auth to a model, is safe to pass ingress auth `nil`.
func mapToModel(ing *networkingv1beta1.Ingress, ia *authv1.IngressAuth) model.App {
	app := mapIngressToModel(ing)
	app.ProxySettings = mapIngressAuthToModel(ia)

	return app
}

// mapIngressToModel maps the base data of the app, this data is obtained from the ingress.
func mapIngressToModel(ing *networkingv1beta1.Ingress) model.App {
	return model.App{
		ID:            fmt.Sprintf("%s/%s", ing.Namespace, ing.Name),
		AuthBackendID: ing.Annotations[backendAnnotation],
		Host:          ing.Spec.Rules[0].Host,
		Ingress: model.KubernetesIngress{
			Name:      ing.Name,
			Namespace: ing.Namespace,
			Upstream: model.KubernetesService{
				Name:           ing.Spec.Rules[0].HTTP.Paths[0].Backend.ServiceName,
				Namespace:      ing.Namespace,
				PortOrPortName: ing.Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort.String(),
			},
		},
	}
}

// mapIngressAuthToModel maps proxy settings based data.
// TODO(slok): Load defaults from cmd start.
func mapIngressAuthToModel(ia *authv1.IngressAuth) model.ProxySettings {
	if ia == nil {
		return model.ProxySettings{}
	}

	// Set global proxy settings.
	ps := model.ProxySettings{
		Scopes: ia.Spec.AuthSettings.ScopeOrClaims,
	}

	// Set specific proxy settings.
	switch {

	case ia.Spec.AuthProxySource.Oauth2Proxy != nil:
		ps.Oauth2Proxy = &model.Oauth2ProxySettings{
			Image:     ia.Spec.AuthProxySource.Oauth2Proxy.Image,
			Replicas:  ia.Spec.AuthProxySource.Oauth2Proxy.Replicas,
			Resources: ia.Spec.AuthProxySource.Oauth2Proxy.Resources,
		}
	}

	return ps
}

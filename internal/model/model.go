package model

import (
	corev1 "k8s.io/api/core/v1"
)

// AuthBackend is the backend that has the auth system.
type AuthBackend struct {
	ID string

	Dex *AuthBackendDex
}

// AuthBackendDex is the configuration of dex AuthBackend.
type AuthBackendDex struct {
	APIURL    string
	PublicURL string
}

// App is a representation of an app that wants to be secured.
type App struct {
	ID            string
	AuthBackendID string
	Host          string
	Ingress       KubernetesIngress
	ProxySettings ProxySettings
}

// KubernetesIngress is the kubernetes service related to the App.
type KubernetesIngress struct {
	Name      string
	Namespace string
	Upstream  KubernetesService
}

// KubernetesService is the kubernetes service related to the App.
type KubernetesService struct {
	Name           string
	Namespace      string
	PortOrPortName string
}

// ProxySettings settings are the settings of an oauth2-proxy.
type ProxySettings struct {
	Scopes      []string
	Oauth2Proxy *Oauth2ProxySettings
}

// Oauth2ProxySettings are the settings for an oauth2proxy.
type Oauth2ProxySettings struct {
	Image     string
	Replicas  int
	Resources *corev1.ResourceRequirements // Stable and core (in K8s) enough type to accept as a valid app model type.
}

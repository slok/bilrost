package model

// AuthBackend is the backend that has the auth system.
type AuthBackend struct {
	ID string

	Dex *AuthBackendDex
}

// AuthBackendDex is the configuraiton of dex AuthBackend.
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
}

// KubernetesIngress is the kubernetes service related to the App
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

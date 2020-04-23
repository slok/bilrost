package proxy

import (
	"context"
)

// OIDCProxySettings are the settings of the proxy.
type OIDCProxySettings struct {
	// URL is the Public URL where the app is listening.
	URL string
	// UpstreamURL is the internal URL where the app is litening.
	UpstreamURL string
	//IssuerURL is the public URL where the auth service is issuing the tokens (e.g Dex public URL).
	IssuerURL string
	// ClientID is the id that identifies the app in the auth service.
	ClientID string
	// ClientSecret is the secret used for the app to communicate with the auth service.
	ClientSecret string
	// Scopes are the Oauth/OIDC scopes asked to the auth service to set that info in the token.
	Scopes []string
	// IngressName is the app's ingress, the entrypoint to the application that we must secure.
	IngressName string
	// IngressNamespace is the namespace where the app's ingress is living.
	IngressNamespace string
}

// UnprovisionSettings are the settings that the proxy service needs to restore
// to the previous state.
type UnprovisionSettings struct {
	IngressName                   string
	IngressNamespace              string
	OriginalServiceName           string
	OriginalServicePortOrNamePort string
}

// OIDCProvisioner knows how to provision an OIDC proxy to be able
// to connect the proxy with the app as upstream and the
// auth backend as the authentication service.
type OIDCProvisioner interface {
	Provision(ctx context.Context, settings OIDCProxySettings) error
	Unprovision(ctx context.Context, settings UnprovisionSettings) error
}

//go:generate mockery -case underscore -output proxymock -outpkg proxymock -name OIDCProvisioner

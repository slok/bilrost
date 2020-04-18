package proxy

import "context"

// OIDCProxySettings are the settings of the proxy.
type OIDCProxySettings struct {
	URL         string
	UpstreamURL string
	IssuerURL   string
	AppID       string
	AppSecret   string
	Scopes      []string
}

// OIDCProvisioner knows how to provision an OIDC proxy to be able
// to connect the proxy with the app as upstream and the
// auth backend as the authentication service.
type OIDCProvisioner interface {
	Provision(ctx context.Context, settings OIDCProxySettings) error
	Unprovision(ctx context.Context, settings OIDCProxySettings) error
}

//go:generate mockery -case underscore -output proxymock -outpkg proxymock -name OIDCProvisioner

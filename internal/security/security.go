package security

import (
	"context"
	"fmt"

	"github.com/slok/bilrost/internal/authbackend"
	authbackendfactory "github.com/slok/bilrost/internal/authbackend/factory"
	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/model"
	"github.com/slok/bilrost/internal/proxy"
)

// AuthBackendRepository knows how to get AuthBackends from a storage.
type AuthBackendRepository interface {
	GetAuthBackend(ctx context.Context, id string) (*model.AuthBackend, error)
}

//go:generate mockery -case underscore -output securitymock -outpkg securitymock -name AuthBackendRepository

// KubeServiceTranslator knows how to translate a kubernetes service to a URL.
type KubeServiceTranslator interface {
	GetServiceHostAndPort(ctx context.Context, svc model.KubernetesService) (string, int, error)
}

//go:generate mockery -case underscore -output securitymock -outpkg securitymock -name KubeServiceTranslator

// Service is the application service where all the security of an application
// happens.
type Service interface {
	SecureApp(ctx context.Context, app model.App) error
	RollbackAppSecurity(ctx context.Context, app model.App) error
}

type service struct {
	proxyProvisioner proxy.OIDCProvisioner
	abRepo           AuthBackendRepository
	abRegFactory     authbackend.AppRegistererFactory
	svcTranslator    KubeServiceTranslator
	logger           log.Logger
}

// ServiceConfig is the service configuration.
type ServiceConfig struct {
	ServiceTranslator      KubeServiceTranslator
	OIDCProxyProvisioner   proxy.OIDCProvisioner
	AuthBackendRepo        AuthBackendRepository
	AuthBackendRepoFactory authbackend.AppRegistererFactory
	Logger                 log.Logger
}

func (c *ServiceConfig) defaults() error {
	if c.Logger == nil {
		c.Logger = log.Dummy
	}
	c.Logger = c.Logger.WithKV(log.KV{"service": "security.Service"})

	if c.AuthBackendRepo == nil {
		return fmt.Errorf("auth backends repository is required")
	}

	if c.AuthBackendRepoFactory == nil {
		c.AuthBackendRepoFactory = authbackendfactory.Default
	}

	if c.OIDCProxyProvisioner == nil {
		return fmt.Errorf("an OIDC proxy provisioner is required")
	}

	if c.ServiceTranslator == nil {
		return fmt.Errorf("a Kubernetes service translator is required")
	}

	return nil
}

// NewService returns a new Service implementation.
func NewService(cfg ServiceConfig) (Service, error) {
	err := cfg.defaults()
	if err != nil {
		return nil, fmt.Errorf("configuration is not valid: %w", err)
	}

	return service{
		svcTranslator:    cfg.ServiceTranslator,
		proxyProvisioner: cfg.OIDCProxyProvisioner,
		abRepo:           cfg.AuthBackendRepo,
		abRegFactory:     cfg.AuthBackendRepoFactory,
		logger:           cfg.Logger,
	}, nil
}

func (s service) SecureApp(ctx context.Context, app model.App) error {
	ab, err := s.abRepo.GetAuthBackend(ctx, app.AuthBackendID)
	if err != nil {
		return fmt.Errorf("could not retrieve backend information: %w", err)
	}

	// Get the backend to register the app and register.
	abReg, err := s.abRegFactory.GetAppRegisterer(*ab)
	if err != nil {
		return fmt.Errorf("could not get app backend to register the app")
	}
	oa := authbackend.OIDCApp{
		ID:          app.ID,
		Name:        app.ID,
		CallBackURL: fmt.Sprintf("https://%s/oauth2/callback", app.Host), // TODO(slok): Configurable based on the proxy.
		Secret:      "TODO",                                              // TODO(slok): Need a way of setting a proper secret.
	}
	err = abReg.RegisterApp(ctx, oa)
	if err != nil {
		return fmt.Errorf("could not register oauth application on backend: %w", err)
	}

	// Get Upstream URL.
	host, port, err := s.svcTranslator.GetServiceHostAndPort(ctx, app.Ingress.Upstream)
	if err != nil {
		return fmt.Errorf("could not translate ingress upstream service to host and port: %w", err)
	}

	// Create the proxy.
	abPublicURL := ""
	switch {
	case ab.Dex != nil:
		abPublicURL = ab.Dex.PublicURL
	}
	proxySettings := proxy.OIDCProxySettings{
		URL: fmt.Sprintf("https://%s", app.Host),
		// TODO(slok): Is always http? https?
		UpstreamURL:      fmt.Sprintf("http://%s:%d", host, port),
		IssuerURL:        abPublicURL,
		AppID:            oa.ID,
		AppSecret:        oa.Secret,
		IngressName:      app.Ingress.Name,
		IngressNamespace: app.Ingress.Namespace,
	}
	err = s.proxyProvisioner.Provision(ctx, proxySettings)
	if err != nil {
		return fmt.Errorf("could not provision OIDC proxy: %w", err)
	}

	return nil
}

func (s service) RollbackAppSecurity(ctx context.Context, app model.App) error {
	return fmt.Errorf("not implemented")
}

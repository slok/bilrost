package security

import (
	"context"
	"fmt"

	"github.com/slok/bilrost/internal/authbackend"
	"github.com/slok/bilrost/internal/backup"
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

//go:generate mockery -case underscore -output securitymock -outpkg securitymock -name Service

type service struct {
	backupper        backup.Backupper
	proxyProvisioner proxy.OIDCProvisioner
	abRepo           AuthBackendRepository
	abRegFactory     authbackend.AppRegistererFactory
	svcTranslator    KubeServiceTranslator
	logger           log.Logger
}

// ServiceConfig is the service configuration.
type ServiceConfig struct {
	Backupper             backup.Backupper
	ServiceTranslator     KubeServiceTranslator
	OIDCProxyProvisioner  proxy.OIDCProvisioner
	AuthBackendRepo       AuthBackendRepository
	AuthBackendRegFactory authbackend.AppRegistererFactory
	Logger                log.Logger
}

func (c *ServiceConfig) defaults() error {
	if c.Logger == nil {
		c.Logger = log.Dummy
	}
	c.Logger = c.Logger.WithKV(log.KV{"service": "security.Service"})

	if c.AuthBackendRepo == nil {
		return fmt.Errorf("auth backends repository is required")
	}

	if c.AuthBackendRegFactory == nil {
		return fmt.Errorf("auth backend registerers factory is required")
	}

	if c.OIDCProxyProvisioner == nil {
		return fmt.Errorf("an OIDC proxy provisioner is required")
	}

	if c.ServiceTranslator == nil {
		return fmt.Errorf("a Kubernetes service translator is required")
	}

	if c.Backupper == nil {
		return fmt.Errorf("a backup service is required")
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
		backupper:        cfg.Backupper,
		svcTranslator:    cfg.ServiceTranslator,
		proxyProvisioner: cfg.OIDCProxyProvisioner,
		abRepo:           cfg.AuthBackendRepo,
		abRegFactory:     cfg.AuthBackendRegFactory,
		logger:           cfg.Logger,
	}, nil
}

func (s service) SecureApp(ctx context.Context, app model.App) error {
	ab, err := s.abRepo.GetAuthBackend(ctx, app.AuthBackendID)
	if err != nil {
		return fmt.Errorf("could not retrieve backend information: %w", err)
	}

	// Get the auth backend to register the app and register.
	abReg, err := s.abRegFactory.GetAppRegisterer(*ab)
	if err != nil {
		return fmt.Errorf("could not get app backend to register the app")
	}
	oa := authbackend.OIDCApp{
		ID:          app.ID,
		Name:        app.ID,
		CallBackURL: fmt.Sprintf("https://%s/oauth2/callback", app.Host), // TODO(slok): Configurable based on the proxy.
	}
	oaRes, err := abReg.RegisterApp(ctx, oa)
	if err != nil {
		return fmt.Errorf("could not register oauth application on backend: %w", err)
	}

	// Backup original Ingress service data or load from a previous backup if already there.
	bkData := &backup.Data{
		AuthBackendID:         app.AuthBackendID,
		ServiceName:           app.Ingress.Upstream.Name,
		ServicePortOrNamePort: app.Ingress.Upstream.PortOrPortName,
	}
	bkData, err = s.backupper.BackupOrGet(ctx, app, *bkData)
	if err != nil {
		return fmt.Errorf("could not backup or get backup data: %w", err)
	}
	if bkData != nil {
		app.Ingress.Upstream.Name = bkData.ServiceName
		app.Ingress.Upstream.PortOrPortName = bkData.ServicePortOrNamePort
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
		UpstreamURL:  fmt.Sprintf("http://%s:%d", host, port),
		IssuerURL:    abPublicURL,
		ClientID:     oaRes.ClientID,
		ClientSecret: oaRes.ClientSecret,
		App:          app,
	}
	err = s.proxyProvisioner.Provision(ctx, proxySettings)
	if err != nil {
		return fmt.Errorf("could not provision OIDC proxy: %w", err)
	}

	return nil
}

func (s service) RollbackAppSecurity(ctx context.Context, app model.App) error {
	bkData, err := s.backupper.GetBackup(ctx, app)
	if err != nil {
		return fmt.Errorf("could not get backup data: %w", err)
	}

	// Uprovision proxy.
	proxySettings := proxy.UnprovisionSettings{
		IngressName:                   app.Ingress.Name,
		IngressNamespace:              app.Ingress.Namespace,
		OriginalServiceName:           bkData.ServiceName,
		OriginalServicePortOrNamePort: bkData.ServicePortOrNamePort,
	}
	err = s.proxyProvisioner.Unprovision(ctx, proxySettings)
	if err != nil {
		return fmt.Errorf("could not unprovision OIDC proxy: %w", err)
	}

	// Get the auth backend to unregister the app.
	ab, err := s.abRepo.GetAuthBackend(ctx, bkData.AuthBackendID)
	if err != nil {
		return fmt.Errorf("could not retrieve backend information: %w", err)
	}
	abReg, err := s.abRegFactory.GetAppRegisterer(*ab)
	if err != nil {
		return fmt.Errorf("could not get app backend to register the app")
	}
	err = abReg.UnregisterApp(ctx, app.ID)
	if err != nil {
		return fmt.Errorf("could not unregister oauth application on backend: %w", err)
	}

	// Delete backup.
	err = s.backupper.DeleteBackup(ctx, app)
	if err != nil {
		return fmt.Errorf("could not delete backup: %w", err)
	}

	return nil
}

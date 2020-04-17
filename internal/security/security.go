package security

import (
	"context"
	"fmt"

	"github.com/slok/bilrost/internal/authbackend"
	authbackendfactory "github.com/slok/bilrost/internal/authbackend/factory"
	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/model"
)

// Service is the application service where all the security of an application
// happens.
type Service interface {
	SecureApp(ctx context.Context, app model.App) error
	RollbackAppSecurity(ctx context.Context, app model.App) error
}

type service struct {
	abRepo       AuthBackendRepository
	abRegFactory authbackend.AppRegistererFactory
	logger       log.Logger
}

// ServiceConfig is the service configuration.
type ServiceConfig struct {
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

	return nil
}

// NewService returns a new Service implementation.
func NewService(cfg ServiceConfig) (Service, error) {
	err := cfg.defaults()
	if err != nil {
		return nil, fmt.Errorf("configuration is not valid: %w", err)
	}

	return service{
		abRepo:       cfg.AuthBackendRepo,
		abRegFactory: cfg.AuthBackendRepoFactory,
		logger:       cfg.Logger,
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

	// Create the proxy.

	// Update ingress.

	return nil
}

func (s service) RollbackAppSecurity(ctx context.Context, app model.App) error {
	return fmt.Errorf("not implemented")
}

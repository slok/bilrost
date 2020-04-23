package dex

import (
	"context"
	"fmt"

	dexapi "github.com/dexidp/dex/api"

	"github.com/slok/bilrost/internal/authbackend"
	"github.com/slok/bilrost/internal/log"
)

// Client is the dex client interface.
type Client interface {
	dexapi.DexClient
}

//go:generate mockery -case underscore -output dexmock -outpkg dexmock -name Client

type appRegisterer struct {
	cli    Client
	logger log.Logger
}

// NewAppRegisterer returns a new application registerer for a dex backend.
func NewAppRegisterer(cli Client, logger log.Logger) authbackend.AppRegisterer {
	return appRegisterer{
		cli:    cli,
		logger: logger.WithKV(log.KV{"service": "authbackend.dex.AppRegisterer"}),
	}
}

func (a appRegisterer) RegisterApp(ctx context.Context, app authbackend.OIDCApp) (*authbackend.OIDCAppRegistryData, error) {
	req := &dexapi.CreateClientReq{
		Client: &dexapi.Client{
			Id:     app.ID,
			Name:   app.Name,
			Secret: "TODO",
			RedirectUris: []string{
				app.CallBackURL,
			},
		},
	}
	_, err := a.cli.CreateClient(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("could not register application on Dex: %w", err)
	}

	a.logger.WithKV(log.KV{"app": app.Name, "callbackURL": app.CallBackURL}).
		Infof("app registered as a client on Dex backend")

	return &authbackend.OIDCAppRegistryData{
		ClientID:     req.Client.Id,
		ClientSecret: req.Client.Secret,
	}, nil
}
func (a appRegisterer) UnregisterApp(ctx context.Context, appID string) error {
	req := &dexapi.DeleteClientReq{Id: appID}
	_, err := a.cli.DeleteClient(ctx, req)
	if err != nil {
		return fmt.Errorf("could not unregister application on Dex: %w", err)
	}

	return nil
}

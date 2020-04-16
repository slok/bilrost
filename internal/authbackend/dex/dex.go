package dex

import (
	"context"
	"fmt"

	dexapi "github.com/dexidp/dex/api"

	"github.com/slok/bilrost/internal/authbackend"
)

// Client is the dex client interface.
type Client interface {
	dexapi.DexClient
}

//go:generate mockery -case underscore -output dexmock -outpkg dexmock -name Client

type appRegisterer struct {
	cli Client
}

// NewAppRegisterer returns a new application registerer for a dex backend.
func NewAppRegisterer(cli Client) authbackend.AppRegisterer {
	return appRegisterer{
		cli: cli,
	}
}

func (a appRegisterer) RegisterApp(ctx context.Context, app authbackend.OIDCApp) error {
	req := &dexapi.CreateClientReq{
		Client: &dexapi.Client{
			Id:     app.ID,
			Name:   app.Name,
			Secret: app.Secret,
			RedirectUris: []string{
				app.CallBackURL,
			},
		},
	}
	_, err := a.cli.CreateClient(ctx, req)
	if err != nil {
		return fmt.Errorf("could not register application on Dex: %w", err)
	}

	return nil
}
func (a appRegisterer) UnregisterApp(ctx context.Context, appID string) error {
	req := &dexapi.DeleteClientReq{Id: appID}
	_, err := a.cli.DeleteClient(ctx, req)
	if err != nil {
		return fmt.Errorf("could not unregister application on Dex: %w", err)
	}

	return nil
}

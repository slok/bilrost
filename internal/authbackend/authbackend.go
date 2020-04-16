package authbackend

import (
	"context"

	"github.com/slok/bilrost/internal/model"
)

// OIDCApp is an app that can be registered on different OIDC auth backends.
type OIDCApp struct {
	ID          string
	Name        string
	CallBackURL string
	Secret      string
}

// AppRegisterer knows how to register OIDC apps on backends.
type AppRegisterer interface {
	RegisterApp(ctx context.Context, app OIDCApp) error
	UnregisterApp(ctx context.Context, appID string) error
}

//go:generate mockery -case underscore -output authbackendmock -outpkg authbackendmock -name AppRegisterer

// AppRegistererFactory gets an app registerer based on an auth backend.
type AppRegistererFactory interface {
	GetAppRegisterer(ab model.AuthBackend) (AppRegisterer, error)
}

//go:generate mockery -case underscore -output authbackendmock -outpkg authbackendmock -name AppRegistererFactory

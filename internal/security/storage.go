package security

import (
	"context"

	"github.com/slok/bilrost/internal/model"
)

// AuthBackendRepository knows how to get AuthBackends from a storage.
type AuthBackendRepository interface {
	GetAuthBackend(ctx context.Context, id string) (*model.AuthBackend, error)
}

//go:generate mockery -case underscore -output securitymock -outpkg securitymock -name AuthBackendRepository

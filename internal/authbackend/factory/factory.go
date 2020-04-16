package factory

import (
	"fmt"

	dexapi "github.com/dexidp/dex/api"
	"google.golang.org/grpc"

	"github.com/slok/bilrost/internal/authbackend"
	"github.com/slok/bilrost/internal/authbackend/dex"
	"github.com/slok/bilrost/internal/model"
)

type factory int

// Default is the default auth backend factory.
const Default = factory(0)

func (factory) GetAppRegisterer(ab model.AuthBackend) (authbackend.AppRegisterer, error) {
	switch {
	// Dex client.
	// TODO(slok): Use an internal cache and return lazy?.
	case ab.Dex != nil:
		conn, err := grpc.Dial(ab.Dex.APIURL, grpc.WithInsecure())
		if err != nil {
			return nil, fmt.Errorf("could not create GRPC Dex API client: %w", err)
		}
		dexCli := dexapi.NewDexClient(conn)
		return dex.NewAppRegisterer(dexCli), nil
	}

	return nil, fmt.Errorf("unknown auth backend type")
}

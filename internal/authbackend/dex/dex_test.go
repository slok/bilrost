package dex_test

import (
	"context"
	"fmt"
	"testing"

	dexapi "github.com/dexidp/dex/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/slok/bilrost/internal/authbackend"
	"github.com/slok/bilrost/internal/authbackend/dex"
	"github.com/slok/bilrost/internal/authbackend/dex/dexmock"
	"github.com/slok/bilrost/internal/log"
)

func TestRegisterApp(t *testing.T) {
	tests := map[string]struct {
		oidcApp authbackend.OIDCApp
		mock    func(c *dexmock.Client)
		expRes  authbackend.OIDCAppRegistryData
		expErr  bool
	}{
		"An error registering the oidc app should be propagated.": {
			oidcApp: authbackend.OIDCApp{},
			mock: func(c *dexmock.Client) {
				c.On("CreateClient", mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Registering correctly the app should call dex client.": {
			oidcApp: authbackend.OIDCApp{
				ID:          "test-id",
				Name:        "test",
				CallBackURL: "https://whatever.dev/oauth2/callback",
			},
			mock: func(c *dexmock.Client) {
				expReq := &dexapi.CreateClientReq{
					Client: &dexapi.Client{
						Id:           "test-id",
						Name:         "test",
						Secret:       "TODO",
						RedirectUris: []string{"https://whatever.dev/oauth2/callback"},
					},
				}
				c.On("CreateClient", mock.Anything, expReq).Once().Return(nil, nil)
			},
			expRes: authbackend.OIDCAppRegistryData{
				ClientID:     "test-id",
				ClientSecret: "TODO",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			mdex := &dexmock.Client{}
			test.mock(mdex)

			ar := dex.NewAppRegisterer(mdex, log.Dummy)
			res, err := ar.RegisterApp(context.TODO(), test.oidcApp)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expRes, *res)
				mdex.AssertExpectations(t)
			}
		})
	}
}

func TestUnregisterApp(t *testing.T) {
	tests := map[string]struct {
		id     string
		mock   func(c *dexmock.Client)
		expErr bool
	}{
		"An error unregistering the oidc app should be propagated.": {
			id: "test-id",
			mock: func(c *dexmock.Client) {
				c.On("DeleteClient", mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Unregistering correctly the app should call dex client.": {
			id: "test-id",
			mock: func(c *dexmock.Client) {
				expReq := &dexapi.DeleteClientReq{Id: "test-id"}
				c.On("DeleteClient", mock.Anything, expReq).Once().Return(nil, nil)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			mdex := &dexmock.Client{}
			test.mock(mdex)

			ar := dex.NewAppRegisterer(mdex, log.Dummy)
			err := ar.UnregisterApp(context.TODO(), test.id)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.NoError(err)
				mdex.AssertExpectations(t)
			}
		})
	}
}

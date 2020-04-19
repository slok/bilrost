package security_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/slok/bilrost/internal/authbackend"
	"github.com/slok/bilrost/internal/authbackend/authbackendmock"
	"github.com/slok/bilrost/internal/model"
	"github.com/slok/bilrost/internal/proxy"
	"github.com/slok/bilrost/internal/proxy/proxymock"
	"github.com/slok/bilrost/internal/security"
	"github.com/slok/bilrost/internal/security/securitymock"
)

func TestSecureApp(t *testing.T) {
	type testMocks struct {
		abRepo        *securitymock.AuthBackendRepository
		abAppReg      *authbackendmock.AppRegisterer
		abAppRegFact  *authbackendmock.AppRegistererFactory
		oidcProxyProv *proxymock.OIDCProvisioner
	}

	tests := map[string]struct {
		app    model.App
		mock   func(m testMocks)
		expErr bool
	}{
		"A correct secured app with Dex backend should make all the steps correctly.": {
			app: model.App{
				ID:            "test-ns/my-app",
				AuthBackendID: "test-ns-dex-backend",
				Host:          "my.app.slok.dev",
				UpstreamURL:   "http://internal-app.svc.cluster.local",
			},
			mock: func(m testMocks) {
				// Get the backend information.
				ab := &model.AuthBackend{
					ID: "test-dex",
					Dex: &model.AuthBackendDex{
						APIURL:    "internal.cluster.url:81",
						PublicURL: "https://test-dex.dev",
					},
				}
				m.abRepo.On("GetAuthBackend", mock.Anything, "test-ns-dex-backend").Once().Return(ab, nil)

				// The app should be registered.
				expOIDCApp := authbackend.OIDCApp{
					ID:          "test-ns/my-app",
					Name:        "test-ns/my-app",
					CallBackURL: "https://my.app.slok.dev/oauth2/callback",
					Secret:      "TODO",
				}
				m.abAppReg.On("RegisterApp", mock.Anything, expOIDCApp).Once().Return(nil)

				// The proxy should be provisioned.
				expProxySettings := proxy.OIDCProxySettings{
					URL:         "https://my.app.slok.dev",
					UpstreamURL: "http://internal-app.svc.cluster.local",
					IssuerURL:   "https://test-dex.dev",
					AppID:       "test-ns/my-app",
					AppSecret:   "TODO",
				}
				m.oidcProxyProv.On("Provision", mock.Anything, expProxySettings).Once().Return(nil)
			},
		},

		"Failing while getting the auth backend shoult stop the process with failure.": {
			mock: func(m testMocks) {
				m.abRepo.On("GetAuthBackend", mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing while registering the app on the auth backend should stop the process with failure.": {
			mock: func(m testMocks) {
				m.abRepo.On("GetAuthBackend", mock.Anything, mock.Anything).Once().Return(&model.AuthBackend{}, nil)
				m.abAppReg.On("RegisterApp", mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing while provisioning the proxy should stop the process with failure.": {
			mock: func(m testMocks) {
				m.abRepo.On("GetAuthBackend", mock.Anything, mock.Anything).Once().Return(&model.AuthBackend{}, nil)
				m.abAppReg.On("RegisterApp", mock.Anything, mock.Anything).Once().Return(nil)
				m.oidcProxyProv.On("Provision", mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Mocks.
			m := testMocks{
				abRepo:        &securitymock.AuthBackendRepository{},
				abAppReg:      &authbackendmock.AppRegisterer{},
				abAppRegFact:  &authbackendmock.AppRegistererFactory{},
				oidcProxyProv: &proxymock.OIDCProvisioner{},
			}
			m.abAppRegFact.On("GetAppRegisterer", mock.Anything).Return(m.abAppReg, nil)
			test.mock(m)

			// Execute.
			cfg := security.ServiceConfig{
				AuthBackendRepo:        m.abRepo,
				AuthBackendRepoFactory: m.abAppRegFact,
				OIDCProxyProvisioner:   m.oidcProxyProv,
			}
			svc, err := security.NewService(cfg)
			require.NoError(err)

			err = svc.SecureApp(context.TODO(), test.app)

			// check
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				m.abRepo.AssertExpectations(t)
				m.abAppReg.AssertExpectations(t)
				m.abAppRegFact.AssertExpectations(t)
				m.oidcProxyProv.AssertExpectations(t)
			}
		})
	}
}

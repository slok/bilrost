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
	"github.com/slok/bilrost/internal/security"
	"github.com/slok/bilrost/internal/security/securitymock"
)

func TestSecureApp(t *testing.T) {
	tests := map[string]struct {
		app    model.App
		mock   func(abrm *securitymock.AuthBackendRepository, abm *authbackendmock.AppRegisterer)
		expErr bool
	}{
		"A correct secured app should make all the steps correctly.": {
			app: model.App{
				ID:            "test-ns/my-app",
				AuthBackendID: "test-ns-dex-backend",
				Host:          "my.app.slok.dev",
			},
			mock: func(abRepo *securitymock.AuthBackendRepository, abAppReg *authbackendmock.AppRegisterer) {
				// Get the backend information.
				abRepo.On("GetAuthBackend", mock.Anything, "test-ns-dex-backend").Once().Return(&model.AuthBackend{}, nil)

				// The app should be registered.
				expOIDCApp := authbackend.OIDCApp{
					ID:          "test-ns/my-app",
					Name:        "test-ns/my-app",
					CallBackURL: "https://my.app.slok.dev/oauth2/callback",
					Secret:      "TODO",
				}
				abAppReg.On("RegisterApp", mock.Anything, expOIDCApp).Once().Return(nil)
			},
		},

		"Failing while getting the auth backend shoult stop the process with failure.": {
			mock: func(abRepo *securitymock.AuthBackendRepository, abAppReg *authbackendmock.AppRegisterer) {
				abRepo.On("GetAuthBackend", mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing while Registering the app on the auth backend shoult stop the process with failure.": {
			mock: func(abRepo *securitymock.AuthBackendRepository, abAppReg *authbackendmock.AppRegisterer) {
				abRepo.On("GetAuthBackend", mock.Anything, mock.Anything).Once().Return(&model.AuthBackend{}, nil)
				abAppReg.On("RegisterApp", mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Mocks.
			abrm := &securitymock.AuthBackendRepository{}
			abfm := &authbackendmock.AppRegistererFactory{}
			abm := &authbackendmock.AppRegisterer{}
			abfm.On("GetAppRegisterer", mock.Anything).Return(abm, nil)
			test.mock(abrm, abm)

			// Execute.
			cfg := security.ServiceConfig{
				AuthBackendRepo:        abrm,
				AuthBackendRepoFactory: abfm,
			}
			svc, err := security.NewService(cfg)
			require.NoError(err)

			err = svc.SecureApp(context.TODO(), test.app)

			// check
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				abrm.AssertExpectations(t)
				abfm.AssertExpectations(t)
				abm.AssertExpectations(t)
			}
		})
	}
}

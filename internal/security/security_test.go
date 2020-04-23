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
	"github.com/slok/bilrost/internal/backup"
	"github.com/slok/bilrost/internal/backup/backupmock"
	"github.com/slok/bilrost/internal/model"
	"github.com/slok/bilrost/internal/proxy"
	"github.com/slok/bilrost/internal/proxy/proxymock"
	"github.com/slok/bilrost/internal/security"
	"github.com/slok/bilrost/internal/security/securitymock"
)

type testMocks struct {
	backupper     *backupmock.Backupper
	svcTranslator *securitymock.KubeServiceTranslator
	abRepo        *securitymock.AuthBackendRepository
	abAppReg      *authbackendmock.AppRegisterer
	abAppRegFact  *authbackendmock.AppRegistererFactory
	oidcProxyProv *proxymock.OIDCProvisioner
}

func TestSecureApp(t *testing.T) {
	tests := map[string]struct {
		app    model.App
		mock   func(m testMocks)
		expErr bool
	}{
		"A new unsecured app with Dex backend should make all the steps correctly to be secured.": {
			app: model.App{
				ID:            "test-ns/my-app",
				AuthBackendID: "test-ns-dex-backend",
				Host:          "my.app.slok.dev",
				Ingress: model.KubernetesIngress{
					Name:      "my-app",
					Namespace: "test-ns",
					Upstream: model.KubernetesService{
						Name:           "internal-app",
						Namespace:      "test-ns",
						PortOrPortName: "http",
					},
				},
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
				}
				oidcAppReg := &authbackend.OIDCAppRegistryData{
					ClientID:     "app1",
					ClientSecret: "my5cr37",
				}
				m.abAppReg.On("RegisterApp", mock.Anything, expOIDCApp).Once().Return(oidcAppReg, nil)

				// The original information should be backup up.
				expData := backup.Data{
					AuthBackendID:         "test-ns-dex-backend",
					ServiceName:           "internal-app",
					ServicePortOrNamePort: "http",
				}
				m.backupper.On("BackupOrGet", mock.Anything, mock.Anything, expData).Once().Return(nil, nil)

				// The service should be translated to URL.
				expSvc := model.KubernetesService{
					Name:           "internal-app",
					Namespace:      "test-ns",
					PortOrPortName: "http",
				}
				m.svcTranslator.On("GetServiceHostAndPort", mock.Anything, expSvc).Once().Return("internal-app.my-ns.svc.cluster.local", 8080, nil)

				// The proxy should be provisioned.
				expProxySettings := proxy.OIDCProxySettings{
					URL:              "https://my.app.slok.dev",
					UpstreamURL:      "http://internal-app.my-ns.svc.cluster.local:8080",
					IssuerURL:        "https://test-dex.dev",
					ClientID:         "app1",
					ClientSecret:     "my5cr37",
					IngressName:      "my-app",
					IngressNamespace: "test-ns",
				}
				m.oidcProxyProv.On("Provision", mock.Anything, expProxySettings).Once().Return(nil)
			},
		},

		"An already secured app with Dex backend should make all the steps correctly to maintain secured.": {
			app: model.App{
				ID:            "test-ns/my-app",
				AuthBackendID: "test-ns-dex-backend",
				Host:          "my.app.slok.dev",
				Ingress: model.KubernetesIngress{
					Name:      "my-app",
					Namespace: "test-ns",
					Upstream: model.KubernetesService{
						Name:           "internal-app-already-secured",
						Namespace:      "test-ns",
						PortOrPortName: "80",
					},
				},
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
				}
				oidcAppReg := &authbackend.OIDCAppRegistryData{
					ClientID:     "app1",
					ClientSecret: "my5cr37",
				}
				m.abAppReg.On("RegisterApp", mock.Anything, expOIDCApp).Once().Return(oidcAppReg, nil)

				// The original information is already there, we return the original upstream.
				expData := backup.Data{
					AuthBackendID:         "test-ns-dex-backend",
					ServiceName:           "internal-app-already-secured",
					ServicePortOrNamePort: "80",
				}
				storedData := backup.Data{
					ServiceName:           "internal-app",
					ServicePortOrNamePort: "http",
				}
				m.backupper.On("BackupOrGet", mock.Anything, mock.Anything, expData).Once().Return(&storedData, nil)

				// The service should be translated to URL.
				expSvc := model.KubernetesService{
					Name:           "internal-app",
					Namespace:      "test-ns",
					PortOrPortName: "http",
				}
				m.svcTranslator.On("GetServiceHostAndPort", mock.Anything, expSvc).Once().Return("internal-app.my-ns.svc.cluster.local", 8080, nil)

				// The proxy should be provisioned.
				expProxySettings := proxy.OIDCProxySettings{
					URL:              "https://my.app.slok.dev",
					UpstreamURL:      "http://internal-app.my-ns.svc.cluster.local:8080",
					IssuerURL:        "https://test-dex.dev",
					ClientID:         "app1",
					ClientSecret:     "my5cr37",
					IngressName:      "my-app",
					IngressNamespace: "test-ns",
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
				m.abAppReg.On("RegisterApp", mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing while backuping the data should stop the process with failure.": {
			mock: func(m testMocks) {
				m.abRepo.On("GetAuthBackend", mock.Anything, mock.Anything).Once().Return(&model.AuthBackend{}, nil)
				m.abAppReg.On("RegisterApp", mock.Anything, mock.Anything).Once().Return(&authbackend.OIDCAppRegistryData{}, nil)
				m.backupper.On("BackupOrGet", mock.Anything, mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing while translating the service to a URL should stop the process with failure.": {
			mock: func(m testMocks) {
				m.abRepo.On("GetAuthBackend", mock.Anything, mock.Anything).Once().Return(&model.AuthBackend{}, nil)
				m.abAppReg.On("RegisterApp", mock.Anything, mock.Anything).Once().Return(&authbackend.OIDCAppRegistryData{}, nil)
				m.backupper.On("BackupOrGet", mock.Anything, mock.Anything, mock.Anything).Once().Return(nil, nil)
				m.svcTranslator.On("GetServiceHostAndPort", mock.Anything, mock.Anything).Once().Return("", 0, fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing while provisioning the proxy should stop the process with failure.": {
			mock: func(m testMocks) {
				m.abRepo.On("GetAuthBackend", mock.Anything, mock.Anything).Once().Return(&model.AuthBackend{}, nil)
				m.abAppReg.On("RegisterApp", mock.Anything, mock.Anything).Once().Return(&authbackend.OIDCAppRegistryData{}, nil)
				m.backupper.On("BackupOrGet", mock.Anything, mock.Anything, mock.Anything).Once().Return(nil, nil)
				m.svcTranslator.On("GetServiceHostAndPort", mock.Anything, mock.Anything).Once().Return("", 0, nil)
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
				backupper:     &backupmock.Backupper{},
				svcTranslator: &securitymock.KubeServiceTranslator{},
				abRepo:        &securitymock.AuthBackendRepository{},
				abAppReg:      &authbackendmock.AppRegisterer{},
				abAppRegFact:  &authbackendmock.AppRegistererFactory{},
				oidcProxyProv: &proxymock.OIDCProvisioner{},
			}
			m.abAppRegFact.On("GetAppRegisterer", mock.Anything).Return(m.abAppReg, nil)
			test.mock(m)

			// Execute.
			cfg := security.ServiceConfig{
				Backupper:              m.backupper,
				ServiceTranslator:      m.svcTranslator,
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
				m.svcTranslator.AssertExpectations(t)
				m.backupper.AssertExpectations(t)
			}
		})
	}
}

func TestRollbackAppSecurity(t *testing.T) {
	tests := map[string]struct {
		app    model.App
		mock   func(m testMocks)
		expErr bool
	}{
		"A secured app with should be unsecured correctly.": {
			app: model.App{
				ID:            "test-ns/my-app",
				AuthBackendID: "",
				Host:          "my.app.slok.dev",
				Ingress: model.KubernetesIngress{
					Name:      "my-app",
					Namespace: "test-ns",
					Upstream: model.KubernetesService{
						Name:           "internal-app",
						Namespace:      "test-ns",
						PortOrPortName: "http",
					},
				},
			},
			mock: func(m testMocks) {
				// Get original information.
				expData := &backup.Data{
					AuthBackendID:         "test-ns-dex-backend",
					ServiceName:           "internal-orig-app",
					ServicePortOrNamePort: "http-orig",
				}
				m.backupper.On("GetBackup", mock.Anything, mock.Anything).Once().Return(expData, nil)

				// The proxy should be removed.
				expProxySettings := proxy.UnprovisionSettings{
					IngressName:                   "my-app",
					IngressNamespace:              "test-ns",
					OriginalServiceName:           "internal-orig-app",
					OriginalServicePortOrNamePort: "http-orig",
				}
				m.oidcProxyProv.On("Unprovision", mock.Anything, expProxySettings).Once().Return(nil)

				// Get the backend information.
				ab := &model.AuthBackend{
					ID: "test-dex",
					Dex: &model.AuthBackendDex{
						APIURL:    "internal.cluster.url:81",
						PublicURL: "https://test-dex.dev",
					},
				}
				m.abRepo.On("GetAuthBackend", mock.Anything, "test-ns-dex-backend").Once().Return(ab, nil)

				// The app should be unregistered.
				m.abAppReg.On("UnregisterApp", mock.Anything, "test-ns/my-app").Once().Return(nil)

				// The backup should be deleted.
				m.backupper.On("DeleteBackup", mock.Anything, mock.Anything).Once().Return(nil)
			},
		},

		"Failing while getting the backup should stop the process with failure.": {
			mock: func(m testMocks) {
				m.backupper.On("GetBackup", mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing while unprovisioning the proxy should stop the process with failure.": {
			mock: func(m testMocks) {
				m.backupper.On("GetBackup", mock.Anything, mock.Anything).Once().Return(&backup.Data{}, nil)
				m.oidcProxyProv.On("Unprovision", mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing while getting the backend information should stop the process with failure.": {
			mock: func(m testMocks) {
				m.backupper.On("GetBackup", mock.Anything, mock.Anything).Once().Return(&backup.Data{}, nil)
				m.oidcProxyProv.On("Unprovision", mock.Anything, mock.Anything).Once().Return(nil)
				m.abRepo.On("GetAuthBackend", mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing while unregistering in the auth backend should stop the process with failure.": {
			mock: func(m testMocks) {
				m.backupper.On("GetBackup", mock.Anything, mock.Anything).Once().Return(&backup.Data{}, nil)
				m.oidcProxyProv.On("Unprovision", mock.Anything, mock.Anything).Once().Return(nil)
				m.abRepo.On("GetAuthBackend", mock.Anything, mock.Anything).Once().Return(&model.AuthBackend{}, nil)
				m.abAppReg.On("UnregisterApp", mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Failing while deleting the backup should stop the process with failure.": {
			mock: func(m testMocks) {
				m.backupper.On("GetBackup", mock.Anything, mock.Anything).Once().Return(&backup.Data{}, nil)
				m.oidcProxyProv.On("Unprovision", mock.Anything, mock.Anything).Once().Return(nil)
				m.abRepo.On("GetAuthBackend", mock.Anything, mock.Anything).Once().Return(&model.AuthBackend{}, nil)
				m.abAppReg.On("UnregisterApp", mock.Anything, mock.Anything).Once().Return(nil)
				m.backupper.On("DeleteBackup", mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
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
				backupper:     &backupmock.Backupper{},
				svcTranslator: &securitymock.KubeServiceTranslator{},
				abRepo:        &securitymock.AuthBackendRepository{},
				abAppReg:      &authbackendmock.AppRegisterer{},
				abAppRegFact:  &authbackendmock.AppRegistererFactory{},
				oidcProxyProv: &proxymock.OIDCProvisioner{},
			}
			m.abAppRegFact.On("GetAppRegisterer", mock.Anything).Return(m.abAppReg, nil)
			test.mock(m)

			// Execute.
			cfg := security.ServiceConfig{
				Backupper:              m.backupper,
				ServiceTranslator:      m.svcTranslator,
				AuthBackendRepo:        m.abRepo,
				AuthBackendRepoFactory: m.abAppRegFact,
				OIDCProxyProvisioner:   m.oidcProxyProv,
			}
			svc, err := security.NewService(cfg)
			require.NoError(err)

			err = svc.RollbackAppSecurity(context.TODO(), test.app)

			// check
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				m.abRepo.AssertExpectations(t)
				m.abAppReg.AssertExpectations(t)
				m.abAppRegFact.AssertExpectations(t)
				m.oidcProxyProv.AssertExpectations(t)
				m.svcTranslator.AssertExpectations(t)
				m.backupper.AssertExpectations(t)
			}
		})
	}
}

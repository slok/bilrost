package dex_test

import (
	"context"
	"fmt"
	"testing"

	dexapi "github.com/dexidp/dex/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/bilrost/internal/authbackend"
	"github.com/slok/bilrost/internal/authbackend/dex"
	"github.com/slok/bilrost/internal/authbackend/dex/dexmock"
)

func getBaseApp() authbackend.OIDCApp {
	return authbackend.OIDCApp{
		ID:          "test-id",
		Name:        "test",
		CallBackURL: "https://whatever.dev/oauth2/callback",
	}
}

func getBaseConfig() dex.AppRegistererConfig {
	return dex.AppRegistererConfig{
		RunningNamespace: "test-ns",
		SecretGenerator:  func(_ authbackend.OIDCApp) (string, error) { return "53cr37", nil },
	}
}

func getBaseSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bilrost-dex-cli-361dc45aacd2d2a1961554d12a2d666b",
			Namespace: "test-ns",
			Annotations: map[string]string{
				"bilrost.slok.dev/dex-client-id": "test-id",
			},
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "bilrost",
				"app.kubernetes.io/name":       "bilrost",
				"app.kubernetes.io/component":  "dex-client-data",
				"app.kubernetes.io/instance":   "bilrost-dex-cli-361dc45aacd2d2a1961554d12a2d666b",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"clientSecret": []byte("53cr37"),
		},
	}
}

func getBaseDexCreateRequest() *dexapi.CreateClientReq {
	return &dexapi.CreateClientReq{
		Client: &dexapi.Client{
			Id:           "test-id",
			Name:         "test",
			Secret:       "53cr37",
			RedirectUris: []string{"https://whatever.dev/oauth2/callback"},
		},
	}
}

func getBaseResultData() authbackend.OIDCAppRegistryData {
	return authbackend.OIDCAppRegistryData{
		ClientID:     "test-id",
		ClientSecret: "53cr37",
	}
}

func TestRegisterApp(t *testing.T) {
	tests := map[string]struct {
		config  func() dex.AppRegistererConfig
		oidcApp func() authbackend.OIDCApp
		mock    func(c *dexmock.Client, k *dexmock.KubernetesRepository)
		expRes  func() authbackend.OIDCAppRegistryData
		expErr  bool
	}{

		"An error getting the secret should be propagated.": {
			config:  getBaseConfig,
			oidcApp: getBaseApp,
			mock: func(c *dexmock.Client, k *dexmock.KubernetesRepository) {
				k.On("GetSecret", mock.Anything, mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"An error registering the oidc app should be propagated.": {
			config:  getBaseConfig,
			oidcApp: getBaseApp,
			mock: func(c *dexmock.Client, k *dexmock.KubernetesRepository) {
				expSecret := getBaseSecret()
				k.On("GetSecret", mock.Anything, mock.Anything, mock.Anything).Once().Return(expSecret, nil)

				c.On("CreateClient", mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"An error setting a new secret should be propagated.": {
			config:  getBaseConfig,
			oidcApp: getBaseApp,
			mock: func(c *dexmock.Client, k *dexmock.KubernetesRepository) {
				storedSecret := getBaseSecret()
				delete(storedSecret.Data, "clientSecret") // Force creation.
				k.On("GetSecret", mock.Anything, mock.Anything, mock.Anything).Once().Return(storedSecret, nil)

				expSecret := getBaseSecret()
				k.On("EnsureSecret", mock.Anything, expSecret).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Registering a new app should create a secret and register on Dex.": {
			config:  getBaseConfig,
			oidcApp: getBaseApp,
			mock: func(c *dexmock.Client, k *dexmock.KubernetesRepository) {
				// Ask for the secret on Kubernetes.
				expErr := &kubeerrors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}
				k.On("GetSecret", mock.Anything, "test-ns", "bilrost-dex-cli-361dc45aacd2d2a1961554d12a2d666b").Once().Return(nil, expErr)

				// No secret, should create a new one using our generator.
				expSecret := getBaseSecret()
				k.On("EnsureSecret", mock.Anything, expSecret).Once().Return(nil)

				expDelReq := &dexapi.DeleteClientReq{Id: "test-id"}
				c.On("DeleteClient", mock.Anything, expDelReq).Once().Return(nil, nil)
				expCreReq := getBaseDexCreateRequest()
				c.On("CreateClient", mock.Anything, expCreReq).Once().Return(nil, nil)
			},
			expRes: getBaseResultData,
		},

		"Registering a present app with already the secret created should use the secret and register on Dex.": {
			config:  getBaseConfig,
			oidcApp: getBaseApp,
			mock: func(c *dexmock.Client, k *dexmock.KubernetesRepository) {
				// Ask for the secret on Kubernetes.
				expSecret := getBaseSecret()
				expSecret.Data["clientSecret"] = []byte("old-secret")
				k.On("GetSecret", mock.Anything, "test-ns", "bilrost-dex-cli-361dc45aacd2d2a1961554d12a2d666b").Once().Return(expSecret, nil)

				expReq := getBaseDexCreateRequest()
				expReq.Client.Secret = "old-secret"
				c.On("CreateClient", mock.Anything, expReq).Once().Return(nil, nil)
			},
			expRes: func() authbackend.OIDCAppRegistryData {
				r := getBaseResultData()
				r.ClientSecret = "old-secret"
				return r
			},
		},

		"Registering a present app with already the secret created but empty should overwrite and register on Dex.": {
			config:  getBaseConfig,
			oidcApp: getBaseApp,
			mock: func(c *dexmock.Client, k *dexmock.KubernetesRepository) {
				// Ask for the secret on Kubernetes.
				storedSecret := getBaseSecret()
				storedSecret.Data["clientSecret"] = []byte("") // Force creation.
				k.On("GetSecret", mock.Anything, "test-ns", "bilrost-dex-cli-361dc45aacd2d2a1961554d12a2d666b").Once().Return(storedSecret, nil)

				// Empty secret, should create a new one using our generator.
				expSecret := getBaseSecret()
				k.On("EnsureSecret", mock.Anything, expSecret).Once().Return(nil)

				expDelReq := &dexapi.DeleteClientReq{Id: "test-id"}
				c.On("DeleteClient", mock.Anything, expDelReq).Once().Return(nil, nil)
				expCreReq := getBaseDexCreateRequest()
				c.On("CreateClient", mock.Anything, expCreReq).Once().Return(nil, nil)
			},
			expRes: getBaseResultData,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			mdex := &dexmock.Client{}
			mkr := &dexmock.KubernetesRepository{}
			test.mock(mdex, mkr)

			cfg := test.config()
			cfg.Client = mdex
			cfg.KubernetesRepository = mkr
			ar, err := dex.NewAppRegisterer(cfg)
			require.NoError(err)
			res, err := ar.RegisterApp(context.TODO(), test.oidcApp())

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expRes(), *res)
				mdex.AssertExpectations(t)
			}
		})
	}
}

func TestUnregisterApp(t *testing.T) {
	tests := map[string]struct {
		config func() dex.AppRegistererConfig
		id     string
		mock   func(c *dexmock.Client, k *dexmock.KubernetesRepository)
		expErr bool
	}{

		"An error unregistering the oidc app should be propagated.": {
			config: getBaseConfig,
			id:     "test-id",
			mock: func(c *dexmock.Client, k *dexmock.KubernetesRepository) {
				c.On("DeleteClient", mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"An error deleting the oidc app data should be propagated.": {
			config: getBaseConfig,
			id:     "test-id",
			mock: func(c *dexmock.Client, k *dexmock.KubernetesRepository) {
				c.On("DeleteClient", mock.Anything, mock.Anything).Once().Return(nil, nil)
				k.On("DeleteSecret", mock.Anything, mock.Anything, mock.Anything).Once().Return(fmt.Errorf("wanted error"))
			},
			expErr: true,
		},

		"Unregistering correctly the app should call dex client and delete the app data.": {
			config: getBaseConfig,
			id:     "test-id",
			mock: func(c *dexmock.Client, k *dexmock.KubernetesRepository) {
				expReq := &dexapi.DeleteClientReq{Id: "test-id"}
				c.On("DeleteClient", mock.Anything, expReq).Once().Return(nil, nil)
				k.On("DeleteSecret", mock.Anything, "test-ns", "bilrost-dex-cli-361dc45aacd2d2a1961554d12a2d666b").Once().Return(nil)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			mdex := &dexmock.Client{}
			mkr := &dexmock.KubernetesRepository{}
			test.mock(mdex, mkr)

			cfg := test.config()
			cfg.Client = mdex
			cfg.KubernetesRepository = mkr
			ar, err := dex.NewAppRegisterer(cfg)
			require.NoError(err)
			err = ar.UnregisterApp(context.TODO(), test.id)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.NoError(err)
				mdex.AssertExpectations(t)
			}
		})
	}
}

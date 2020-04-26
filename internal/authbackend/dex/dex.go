package dex

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	dexapi "github.com/dexidp/dex/api"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/bilrost/internal/authbackend"
	"github.com/slok/bilrost/internal/log"
)

// Client is the dex client interface.
type Client interface {
	CreateClient(ctx context.Context, in *dexapi.CreateClientReq, opts ...grpc.CallOption) (*dexapi.CreateClientResp, error)
	DeleteClient(ctx context.Context, in *dexapi.DeleteClientReq, opts ...grpc.CallOption) (*dexapi.DeleteClientResp, error)
}

//go:generate mockery -case underscore -output dexmock -outpkg dexmock -name Client

// KubernetesRepository is the service used by dex registerer to interact with k8s.
type KubernetesRepository interface {
	EnsureSecret(ctx context.Context, sec *corev1.Secret) error
	GetSecret(ctx context.Context, ns, name string) (*corev1.Secret, error)
	DeleteSecret(ctx context.Context, ns, name string) error
}

//go:generate mockery -case underscore -output dexmock -outpkg dexmock -name KubernetesRepository

// AppRegistererConfig is the configuration for the app registerer.
type AppRegistererConfig struct {
	RunningNamespace     string
	Client               Client
	KubernetesRepository KubernetesRepository
	SecretGenerator      func(app authbackend.OIDCApp) (string, error)
	Logger               log.Logger
}

func (c *AppRegistererConfig) defaults() error {
	if c.RunningNamespace == "" {
		return fmt.Errorf("the namespace where the app is running is required")
	}

	if c.Client == nil {
		return fmt.Errorf("a Dex client is required")
	}

	if c.KubernetesRepository == nil {
		return fmt.Errorf("a Kubernetes repository required")
	}

	if c.SecretGenerator == nil {
		c.SecretGenerator = func(_ authbackend.OIDCApp) (string, error) {
			b := make([]byte, 32)
			_, err := rand.Read(b)
			if err != nil {
				return "", fmt.Errorf("could not generate random password: %w", err)
			}

			return base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
		}
	}

	if c.Logger == nil {
		c.Logger = log.Dummy
	}
	c.Logger = c.Logger.WithKV(log.KV{"service": "authbackend.dex.AppRegisterer"})

	return nil
}

type appRegisterer struct {
	secretGenerator  func(app authbackend.OIDCApp) (string, error)
	runningNamespace string
	cli              Client
	kuberepo         KubernetesRepository
	logger           log.Logger
}

// NewAppRegisterer returns a new application registerer for a dex backend.
func NewAppRegisterer(config AppRegistererConfig) (authbackend.AppRegisterer, error) {
	err := config.defaults()
	if err != nil {
		return nil, fmt.Errorf("could not create app registerer: %w", err)
	}

	return appRegisterer{
		secretGenerator:  config.SecretGenerator,
		runningNamespace: config.RunningNamespace,
		cli:              config.Client,
		kuberepo:         config.KubernetesRepository,
		logger:           config.Logger,
	}, nil
}

func (a appRegisterer) RegisterApp(ctx context.Context, app authbackend.OIDCApp) (*authbackend.OIDCAppRegistryData, error) {
	secret, changed, err := a.getAndCreateSecret(ctx, app)
	if err != nil {
		return nil, fmt.Errorf("could not get '%s' app OIDC Dex secret: %w", app.ID, err)
	}

	// changed means that we have a new secret and we need to recreate the client on Dex.
	err = a.registerOnDex(ctx, app, secret, changed)
	if err != nil {
		return nil, fmt.Errorf("could not register app on dex: %w", err)
	}

	a.logger.WithKV(log.KV{"app": app.Name, "callbackURL": app.CallBackURL}).
		Infof("app registered as a client on Dex backend")

	return &authbackend.OIDCAppRegistryData{
		ClientID:     app.ID,
		ClientSecret: secret,
	}, nil
}

// getAndCreateSecret will try getting the OIDC app client secret from a kubernetes secret
// if the secret does not exists or is empty it will generate a new one.
// in case we generated a new secret it will return true on the `changed` flag.
func (a appRegisterer) getAndCreateSecret(ctx context.Context, app authbackend.OIDCApp) (secret string, changed bool, err error) {
	const clientSecretKey = "clientSecret"

	// Check if we already have a secret.
	name := getSecretName(app.ID)
	kSecret, err := a.kuberepo.GetSecret(ctx, a.runningNamespace, name)
	if err == nil {
		secret := string(kSecret.Data[clientSecretKey])
		// If we have the secret, then we don't need to create a new one.
		if secret != "" {
			return secret, false, nil
		}
		// Continue because we have the secret but is empty.
	} else {
		if !kubeerrors.IsNotFound(err) {
			return "", false, err
		}
		// Continue because the secret is not present on Kubernetes.
	}

	// If we reached here means that we need a new secret.
	generatedSecret, err := a.secretGenerator(app)
	if err != nil {
		return "", false, err
	}
	a.logger.Debugf("new secret generated for client '%s'", app.ID)

	// Ensure secret (Create or update).
	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: a.runningNamespace,
			Annotations: map[string]string{
				"bilrost.slok.dev/dex-client-id": app.ID,
			},
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "bilrost",
				"app.kubernetes.io/name":       "bilrost",
				"app.kubernetes.io/component":  "dex-client-data",
				"app.kubernetes.io/instance":   name,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			clientSecretKey: []byte(generatedSecret),
		},
	}

	err = a.kuberepo.EnsureSecret(ctx, newSecret)
	if err != nil {
		return "", false, err
	}

	return generatedSecret, true, nil
}

// registerOnDex will register the application on Dex.
// We have an optional flag called recreate, this is used to delete the client
// prior registering. this is because:
//
// Dex doesn't update the client secret if the app already exists so if we create
// always the client doesn't matter if we had created a new secret, this will be
// ignored by dex and we will end with inconsistencies (with Dex having an old secret
// for the client). So this flag will be used when we want recreate the cleint
func (a appRegisterer) registerOnDex(ctx context.Context, app authbackend.OIDCApp, secret string, recreate bool) error {
	if recreate {
		req := &dexapi.DeleteClientReq{Id: app.ID}
		_, err := a.cli.DeleteClient(ctx, req)
		if err != nil {
			return fmt.Errorf("could not delete client on Dex: %w", err)
		}
	}

	req := &dexapi.CreateClientReq{
		Client: &dexapi.Client{
			Id:     app.ID,
			Name:   app.Name,
			Secret: secret,
			RedirectUris: []string{
				app.CallBackURL,
			},
		},
	}
	_, err := a.cli.CreateClient(ctx, req)
	if err != nil {
		return fmt.Errorf("could not create client on Dex: %w", err)
	}

	return nil
}

func (a appRegisterer) UnregisterApp(ctx context.Context, appID string) error {
	req := &dexapi.DeleteClientReq{Id: appID}
	_, err := a.cli.DeleteClient(ctx, req)
	if err != nil {
		return fmt.Errorf("could not unregister application on Dex: %w", err)
	}

	name := getSecretName(appID)
	err = a.kuberepo.DeleteSecret(ctx, a.runningNamespace, name)
	if err != nil {
		return fmt.Errorf("could not delete '%s' client dex data: %w", appID, err)
	}

	return nil
}

func getSecretName(id string) string {
	checksum := md5.Sum([]byte(id))
	return fmt.Sprintf("bilrost-dex-cli-%x", checksum)
}

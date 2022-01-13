package factory

import (
	"fmt"
	"sync"

	dexapi "github.com/dexidp/dex/api/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/slok/bilrost/internal/authbackend"
	"github.com/slok/bilrost/internal/authbackend/dex"
	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/metrics"
	"github.com/slok/bilrost/internal/model"
)

type factory struct {
	runningNamespace   string
	metricsRecorder    metrics.Recorder
	dexKubeRepo        dex.KubernetesRepository
	appRegisterersPool map[string]authbackend.AppRegisterer
	mu                 sync.Mutex
	logger             log.Logger
}

// NewFactory returns a new authbackend factory.
func NewFactory(runningNamespace string, metricsRecorder metrics.Recorder, dexKubeRepo dex.KubernetesRepository, logger log.Logger) authbackend.AppRegistererFactory {
	return &factory{
		runningNamespace:   runningNamespace,
		metricsRecorder:    metricsRecorder,
		dexKubeRepo:        dexKubeRepo,
		appRegisterersPool: map[string]authbackend.AppRegisterer{},
		logger:             logger,
	}
}

func (f *factory) GetAppRegisterer(ab model.AuthBackend) (authbackend.AppRegisterer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	// Dex client.
	case ab.Dex != nil:
		poolKey := fmt.Sprintf("dex-%s", ab.Dex.APIURL)
		ar, ok := f.appRegisterersPool[poolKey]
		if ok {
			return ar, nil
		}

		// New registerer, store in cache.
		ar, err := f.newDexAppRegisterer(ab)
		if err != nil {
			return nil, err
		}
		f.appRegisterersPool[poolKey] = ar

		return ar, nil
	}

	return nil, fmt.Errorf("unknown auth backend type")
}

func (f *factory) newDexAppRegisterer(ab model.AuthBackend) (authbackend.AppRegisterer, error) {
	conn, err := grpc.Dial(ab.Dex.APIURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("could not create GRPC Dex API client: %w", err)
	}

	cfg := dex.AppRegistererConfig{
		RunningNamespace:     f.runningNamespace,
		KubernetesRepository: f.dexKubeRepo,
		Client:               dex.NewMeasuredClient(f.metricsRecorder, dexapi.NewDexClient(conn)),
		Logger:               f.logger,
	}
	ar, err := dex.NewAppRegisterer(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not create Dex app registerer: %w", err)
	}
	ar = authbackend.NewMeasuredAppRegisterer("dex", f.metricsRecorder, ar)

	return ar, nil
}

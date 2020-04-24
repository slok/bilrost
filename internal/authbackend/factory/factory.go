package factory

import (
	"fmt"

	dexapi "github.com/dexidp/dex/api"
	"google.golang.org/grpc"

	"github.com/slok/bilrost/internal/authbackend"
	"github.com/slok/bilrost/internal/authbackend/dex"
	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/metrics"
	"github.com/slok/bilrost/internal/model"
)

type factory struct {
	runningNamespace string
	metricsRecorder  metrics.Recorder
	dexKubeRepo      dex.KubernetesRepository
	logger           log.Logger
}

// Default is the default auth backend factory.
var Default = factory{logger: log.Dummy}

// NewFactory returns a new authbackend factory.
func NewFactory(runningNamespace string, metricsRecorder metrics.Recorder, dexKubeRepo dex.KubernetesRepository, logger log.Logger) authbackend.AppRegistererFactory {
	return factory{
		runningNamespace: runningNamespace,
		metricsRecorder:  metricsRecorder,
		dexKubeRepo:      dexKubeRepo,
		logger:           logger,
	}
}

func (f factory) GetAppRegisterer(ab model.AuthBackend) (authbackend.AppRegisterer, error) {
	switch {
	// Dex client.
	// TODO(slok): Use an internal cache and return lazy?.
	case ab.Dex != nil:
		conn, err := grpc.Dial(ab.Dex.APIURL, grpc.WithInsecure())
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

	return nil, fmt.Errorf("unknown auth backend type")
}

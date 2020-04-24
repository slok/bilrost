package authbackend

import (
	"context"
	"time"

	"github.com/slok/bilrost/internal/metrics"
)

type measuredAppRegisterer struct {
	appRegType string
	rec        metrics.Recorder
	next       AppRegisterer
}

// NewMeasuredAppRegisterer returns a measured appRegisterer using a metrics.Recorder.
func NewMeasuredAppRegisterer(appRegType string, rec metrics.Recorder, next AppRegisterer) AppRegisterer {
	return measuredAppRegisterer{appRegType: appRegType, rec: rec, next: next}
}

func (m measuredAppRegisterer) RegisterApp(ctx context.Context, app OIDCApp) (o *OIDCAppRegistryData, err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveAuthBackendAppRegistererOperation(ctx, m.appRegType, "RegisterApp", err == nil, t0)
	}(time.Now())
	return m.next.RegisterApp(ctx, app)
}

func (m measuredAppRegisterer) UnregisterApp(ctx context.Context, appID string) (err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveAuthBackendAppRegistererOperation(ctx, m.appRegType, "UnregisterApp", err == nil, t0)
	}(time.Now())
	return m.next.UnregisterApp(ctx, appID)
}

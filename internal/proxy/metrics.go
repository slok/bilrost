package proxy

import (
	"context"
	"time"

	"github.com/slok/bilrost/internal/metrics"
)

type measuredOIDCProvisioner struct {
	provType string
	rec      metrics.Recorder
	next     OIDCProvisioner
}

// NewMeasuredOIDCProvisioner returns a OIDCProvisioner measured using a metrics recorder.
func NewMeasuredOIDCProvisioner(provType string, rec metrics.Recorder, next OIDCProvisioner) OIDCProvisioner {
	return measuredOIDCProvisioner{provType: provType, rec: rec, next: next}
}

func (m measuredOIDCProvisioner) Provision(ctx context.Context, settings OIDCProxySettings) (err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveOIDCProvisionerOperation(ctx, m.provType, "Provision", err == nil, t0)
	}(time.Now())
	return m.next.Provision(ctx, settings)
}

func (m measuredOIDCProvisioner) Unprovision(ctx context.Context, settings UnprovisionSettings) (err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveOIDCProvisionerOperation(ctx, m.provType, "Unprovision", err == nil, t0)
	}(time.Now())

	return m.next.Unprovision(ctx, settings)
}

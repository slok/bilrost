package dex

import (
	"context"
	"time"

	dexapi "github.com/dexidp/dex/api/v2"
	"google.golang.org/grpc"

	"github.com/slok/bilrost/internal/metrics"
)

type measuredClient struct {
	rec  metrics.Recorder
	next Client
}

// NewMeasuredClient returns a Client measured with a metrics recorder.
func NewMeasuredClient(m metrics.Recorder, c Client) Client {
	return measuredClient{rec: m, next: c}
}

func (m measuredClient) CreateClient(ctx context.Context, in *dexapi.CreateClientReq, opts ...grpc.CallOption) (r *dexapi.CreateClientResp, err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveDexAuthBackendDexClientOp(ctx, "CreateClient", err == nil, t0)
	}(time.Now())

	return m.next.CreateClient(ctx, in, opts...)
}

func (m measuredClient) DeleteClient(ctx context.Context, in *dexapi.DeleteClientReq, opts ...grpc.CallOption) (r *dexapi.DeleteClientResp, err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveDexAuthBackendDexClientOp(ctx, "DeleteClient", err == nil, t0)
	}(time.Now())

	return m.next.DeleteClient(ctx, in, opts...)
}

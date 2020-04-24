package prometheus_test

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/slok/bilrost/internal/metrics"
	bilrostprometheus "github.com/slok/bilrost/internal/metrics/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsRecorder(t *testing.T) {
	tests := map[string]struct {
		measure    func(r metrics.Recorder)
		expMetrics []string
	}{
		"Measure Dex client operation duration.": {
			measure: func(r metrics.Recorder) {
				t0 := time.Now()
				ctx := context.TODO()
				r.ObserveDexAuthBackendDexClientOp(ctx, "op1", true, t0.Add(-6*time.Second))
				r.ObserveDexAuthBackendDexClientOp(ctx, "op1", true, t0.Add(-200*time.Millisecond))
				r.ObserveDexAuthBackendDexClientOp(ctx, "op2", false, t0.Add(-600*time.Millisecond))
			},
			expMetrics: []string{
				`# HELP bilrost_dex_auth_backend_dex_client_operation_duration_seconds The duration for an Dex client operation in the dex auth backend.`,
				`# TYPE bilrost_dex_auth_backend_dex_client_operation_duration_seconds histogram`,

				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op1",success="true",le="0.005"} 0`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op1",success="true",le="0.01"} 0`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op1",success="true",le="0.025"} 0`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op1",success="true",le="0.05"} 0`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op1",success="true",le="0.1"} 0`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op1",success="true",le="0.25"} 1`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op1",success="true",le="0.5"} 1`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op1",success="true",le="1"} 1`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op1",success="true",le="2.5"} 1`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op1",success="true",le="5"} 1`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op1",success="true",le="10"} 2`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op1",success="true",le="+Inf"} 2`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_count{operation="op1",success="true"} 2`,

				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op2",success="false",le="0.005"} 0`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op2",success="false",le="0.01"} 0`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op2",success="false",le="0.025"} 0`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op2",success="false",le="0.05"} 0`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op2",success="false",le="0.1"} 0`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op2",success="false",le="0.25"} 0`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op2",success="false",le="0.5"} 0`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op2",success="false",le="1"} 1`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op2",success="false",le="2.5"} 1`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op2",success="false",le="5"} 1`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op2",success="false",le="10"} 1`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_bucket{operation="op2",success="false",le="+Inf"} 1`,
				`bilrost_dex_auth_backend_dex_client_operation_duration_seconds_count{operation="op2",success="false"} 1`,
			},
		},

		"Measure OIDC proxy provisioner operation duration.": {
			measure: func(r metrics.Recorder) {
				t0 := time.Now()
				ctx := context.TODO()
				r.ObserveOIDCProvisionerOperation(ctx, "t1", "op1", true, t0.Add(-6*time.Second))
				r.ObserveOIDCProvisionerOperation(ctx, "t1", "op1", true, t0.Add(-200*time.Millisecond))
				r.ObserveOIDCProvisionerOperation(ctx, "t2", "op2", false, t0.Add(-600*time.Millisecond))
				r.ObserveOIDCProvisionerOperation(ctx, "t2", "op3", false, t0.Add(-75*time.Millisecond))
			},
			expMetrics: []string{
				"# HELP bilrost_oidc_proxy_provisioner_operation_duration_seconds The duration for an OIDC proxy provisioner operation.",
				`# TYPE bilrost_oidc_proxy_provisioner_operation_duration_seconds histogram`,

				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op1",provisioner="t1",success="true",le="0.005"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op1",provisioner="t1",success="true",le="0.01"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op1",provisioner="t1",success="true",le="0.025"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op1",provisioner="t1",success="true",le="0.05"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op1",provisioner="t1",success="true",le="0.1"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op1",provisioner="t1",success="true",le="0.25"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op1",provisioner="t1",success="true",le="0.5"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op1",provisioner="t1",success="true",le="1"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op1",provisioner="t1",success="true",le="2.5"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op1",provisioner="t1",success="true",le="5"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op1",provisioner="t1",success="true",le="10"} 2`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op1",provisioner="t1",success="true",le="+Inf"} 2`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_count{operation="op1",provisioner="t1",success="true"} 2`,

				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op2",provisioner="t2",success="false",le="0.005"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op2",provisioner="t2",success="false",le="0.01"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op2",provisioner="t2",success="false",le="0.025"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op2",provisioner="t2",success="false",le="0.05"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op2",provisioner="t2",success="false",le="0.1"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op2",provisioner="t2",success="false",le="0.25"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op2",provisioner="t2",success="false",le="0.5"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op2",provisioner="t2",success="false",le="1"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op2",provisioner="t2",success="false",le="2.5"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op2",provisioner="t2",success="false",le="5"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op2",provisioner="t2",success="false",le="10"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op2",provisioner="t2",success="false",le="+Inf"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_count{operation="op2",provisioner="t2",success="false"} 1`,

				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op3",provisioner="t2",success="false",le="0.005"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op3",provisioner="t2",success="false",le="0.01"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op3",provisioner="t2",success="false",le="0.025"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op3",provisioner="t2",success="false",le="0.05"} 0`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op3",provisioner="t2",success="false",le="0.1"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op3",provisioner="t2",success="false",le="0.25"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op3",provisioner="t2",success="false",le="0.5"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op3",provisioner="t2",success="false",le="1"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op3",provisioner="t2",success="false",le="2.5"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op3",provisioner="t2",success="false",le="5"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op3",provisioner="t2",success="false",le="10"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_bucket{operation="op3",provisioner="t2",success="false",le="+Inf"} 1`,
				`bilrost_oidc_proxy_provisioner_operation_duration_seconds_count{operation="op3",provisioner="t2",success="false"} 1`,
			},
		},

		"Measure auth backend app registerer operation duration.": {
			measure: func(r metrics.Recorder) {
				t0 := time.Now()
				ctx := context.TODO()
				r.ObserveAuthBackendAppRegistererOperation(ctx, "t1", "op1", true, t0.Add(-6*time.Second))
				r.ObserveAuthBackendAppRegistererOperation(ctx, "t1", "op1", true, t0.Add(-200*time.Millisecond))
				r.ObserveAuthBackendAppRegistererOperation(ctx, "t2", "op2", false, t0.Add(-600*time.Millisecond))
				r.ObserveAuthBackendAppRegistererOperation(ctx, "t2", "op3", false, t0.Add(-75*time.Millisecond))
			},
			expMetrics: []string{
				"# HELP bilrost_auth_backend_app_registerer_operation_duration_seconds The duration for an auth backend app registerer operation.",
				`# TYPE bilrost_auth_backend_app_registerer_operation_duration_seconds histogram`,

				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t1",operation="op1",success="true",le="0.005"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t1",operation="op1",success="true",le="0.01"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t1",operation="op1",success="true",le="0.025"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t1",operation="op1",success="true",le="0.05"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t1",operation="op1",success="true",le="0.1"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t1",operation="op1",success="true",le="0.25"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t1",operation="op1",success="true",le="0.5"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t1",operation="op1",success="true",le="1"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t1",operation="op1",success="true",le="2.5"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t1",operation="op1",success="true",le="5"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t1",operation="op1",success="true",le="10"} 2`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t1",operation="op1",success="true",le="+Inf"} 2`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_count{app_registerer="t1",operation="op1",success="true"} 2`,

				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op2",success="false",le="0.005"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op2",success="false",le="0.01"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op2",success="false",le="0.025"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op2",success="false",le="0.05"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op2",success="false",le="0.1"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op2",success="false",le="0.25"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op2",success="false",le="0.5"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op2",success="false",le="1"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op2",success="false",le="2.5"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op2",success="false",le="5"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op2",success="false",le="10"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op2",success="false",le="+Inf"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_count{app_registerer="t2",operation="op2",success="false"} 1`,

				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op3",success="false",le="0.005"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op3",success="false",le="0.01"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op3",success="false",le="0.025"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op3",success="false",le="0.05"} 0`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op3",success="false",le="0.1"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op3",success="false",le="0.25"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op3",success="false",le="0.5"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op3",success="false",le="1"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op3",success="false",le="2.5"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op3",success="false",le="5"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op3",success="false",le="10"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_bucket{app_registerer="t2",operation="op3",success="false",le="+Inf"} 1`,
				`bilrost_auth_backend_app_registerer_operation_duration_seconds_count{app_registerer="t2",operation="op3",success="false"} 1`,
			},
		},

		"Measure backup backupper operation duration.": {
			measure: func(r metrics.Recorder) {
				t0 := time.Now()
				ctx := context.TODO()
				r.ObserveBackupBackupperOperation(ctx, "t1", "op1", true, t0.Add(-6*time.Second))
				r.ObserveBackupBackupperOperation(ctx, "t1", "op1", true, t0.Add(-200*time.Millisecond))
				r.ObserveBackupBackupperOperation(ctx, "t2", "op2", false, t0.Add(-600*time.Millisecond))
				r.ObserveBackupBackupperOperation(ctx, "t2", "op3", false, t0.Add(-75*time.Millisecond))
			},
			expMetrics: []string{
				`# HELP bilrost_backup_backupper_operation_duration_seconds The duration for a backup backupper operation.`,
				`# TYPE bilrost_backup_backupper_operation_duration_seconds histogram`,

				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t1",operation="op1",success="true",le="0.005"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t1",operation="op1",success="true",le="0.01"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t1",operation="op1",success="true",le="0.025"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t1",operation="op1",success="true",le="0.05"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t1",operation="op1",success="true",le="0.1"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t1",operation="op1",success="true",le="0.25"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t1",operation="op1",success="true",le="0.5"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t1",operation="op1",success="true",le="1"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t1",operation="op1",success="true",le="2.5"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t1",operation="op1",success="true",le="5"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t1",operation="op1",success="true",le="10"} 2`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t1",operation="op1",success="true",le="+Inf"} 2`,
				`bilrost_backup_backupper_operation_duration_seconds_count{backupper="t1",operation="op1",success="true"} 2`,

				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op2",success="false",le="0.005"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op2",success="false",le="0.01"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op2",success="false",le="0.025"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op2",success="false",le="0.05"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op2",success="false",le="0.1"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op2",success="false",le="0.25"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op2",success="false",le="0.5"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op2",success="false",le="1"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op2",success="false",le="2.5"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op2",success="false",le="5"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op2",success="false",le="10"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op2",success="false",le="+Inf"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_count{backupper="t2",operation="op2",success="false"} 1`,

				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op3",success="false",le="0.005"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op3",success="false",le="0.01"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op3",success="false",le="0.025"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op3",success="false",le="0.05"} 0`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op3",success="false",le="0.1"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op3",success="false",le="0.25"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op3",success="false",le="0.5"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op3",success="false",le="1"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op3",success="false",le="2.5"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op3",success="false",le="5"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op3",success="false",le="10"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_bucket{backupper="t2",operation="op3",success="false",le="+Inf"} 1`,
				`bilrost_backup_backupper_operation_duration_seconds_count{backupper="t2",operation="op3",success="false"} 1`,
			},
		},

		"Measure Kubernetes service operation duration.": {
			measure: func(r metrics.Recorder) {
				t0 := time.Now()
				ctx := context.TODO()
				r.ObserveKubernetesServiceOperation(ctx, "ns1", "op1", true, t0.Add(-6*time.Second))
				r.ObserveKubernetesServiceOperation(ctx, "ns1", "op1", true, t0.Add(-200*time.Millisecond))
				r.ObserveKubernetesServiceOperation(ctx, "ns2", "op2", false, t0.Add(-600*time.Millisecond))
				r.ObserveKubernetesServiceOperation(ctx, "ns2", "op3", false, t0.Add(-75*time.Millisecond))
			},
			expMetrics: []string{
				`# HELP bilrost_kubernetes_service_operation_duration_seconds The duration for a kubernetes service operation.`,
				`# TYPE bilrost_kubernetes_service_operation_duration_seconds histogram`,

				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns1",operation="op1",success="true",le="0.005"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns1",operation="op1",success="true",le="0.01"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns1",operation="op1",success="true",le="0.025"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns1",operation="op1",success="true",le="0.05"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns1",operation="op1",success="true",le="0.1"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns1",operation="op1",success="true",le="0.25"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns1",operation="op1",success="true",le="0.5"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns1",operation="op1",success="true",le="1"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns1",operation="op1",success="true",le="2.5"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns1",operation="op1",success="true",le="5"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns1",operation="op1",success="true",le="10"} 2`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns1",operation="op1",success="true",le="+Inf"} 2`,
				`bilrost_kubernetes_service_operation_duration_seconds_count{namespace="ns1",operation="op1",success="true"} 2`,

				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op2",success="false",le="0.005"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op2",success="false",le="0.01"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op2",success="false",le="0.025"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op2",success="false",le="0.05"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op2",success="false",le="0.1"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op2",success="false",le="0.25"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op2",success="false",le="0.5"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op2",success="false",le="1"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op2",success="false",le="2.5"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op2",success="false",le="5"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op2",success="false",le="10"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op2",success="false",le="+Inf"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_count{namespace="ns2",operation="op2",success="false"} 1`,

				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op3",success="false",le="0.005"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op3",success="false",le="0.01"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op3",success="false",le="0.025"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op3",success="false",le="0.05"} 0`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op3",success="false",le="0.1"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op3",success="false",le="0.25"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op3",success="false",le="0.5"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op3",success="false",le="1"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op3",success="false",le="2.5"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op3",success="false",le="5"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op3",success="false",le="10"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_bucket{namespace="ns2",operation="op3",success="false",le="+Inf"} 1`,
				`bilrost_kubernetes_service_operation_duration_seconds_count{namespace="ns2",operation="op3",success="false"} 1`,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Measure.
			reg := prometheus.NewRegistry()
			rec := bilrostprometheus.NewRecorder(reg)
			test.measure(rec)

			// Check.
			h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
			r := httptest.NewRecorder()
			h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/metrics", nil))

			gotMetrics, err := ioutil.ReadAll(r.Body)
			require.NoError(err)
			for _, expMetric := range test.expMetrics {
				assert.Contains(string(gotMetrics), expMetric)
			}
		})
	}
}

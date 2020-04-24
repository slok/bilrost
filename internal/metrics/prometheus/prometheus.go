package prometheus

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	koopercontroller "github.com/spotahome/kooper/controller"
	kooperprometheus "github.com/spotahome/kooper/metrics/prometheus"

	"github.com/slok/bilrost/internal/metrics"
)

// internal type that we make this for two reasons:
// - Avoid collisions when embedding types, this could happen in
// 	 case we embed another interface called `MetricsRecorder`.
// - The external type would be public but we want to be internal
//    to this package.
type kooperRecorder koopercontroller.MetricsRecorder

// recorder knows how to record prometheus metrics in the application.
type recorder struct {
	kooperRecorder

	dexCliOpDuration          *prometheus.HistogramVec
	oidcProxyProvOpDuration   *prometheus.HistogramVec
	authBackAppRegOpDuration  *prometheus.HistogramVec
	backupBackupperOpDuration *prometheus.HistogramVec
	k8sServiceOpDuration      *prometheus.HistogramVec
}

// NewRecorder returns a new Prometheus recorder.
func NewRecorder(reg prometheus.Registerer) metrics.Recorder {
	const (
		promNamespace               = "bilrost"
		promDexCliSubsystem         = "dex_auth_backend_dex_client"
		promProxyProvSubsystem      = "oidc_proxy_provisioner"
		promAuthBackAppRegSubsystem = "auth_backend_app_registerer"
		promBackupperSubsystem      = "backup_backupper"
		promKubernetesSvcSubsystem  = "kubernetes_service"
	)

	r := recorder{
		kooperRecorder: kooperprometheus.New(kooperprometheus.Config{
			Registerer: reg,
		}),

		dexCliOpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: promNamespace,
			Subsystem: promDexCliSubsystem,
			Name:      "operation_duration_seconds",
			Help:      "The duration for an Dex client operation in the dex auth backend.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"operation", "success"}),

		oidcProxyProvOpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: promNamespace,
			Subsystem: promProxyProvSubsystem,
			Name:      "operation_duration_seconds",
			Help:      "The duration for an OIDC proxy provisioner operation.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"provisioner", "operation", "success"}),

		authBackAppRegOpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: promNamespace,
			Subsystem: promAuthBackAppRegSubsystem,
			Name:      "operation_duration_seconds",
			Help:      "The duration for an auth backend app registerer operation.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"app_registerer", "operation", "success"}),

		backupBackupperOpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: promNamespace,
			Subsystem: promBackupperSubsystem,
			Name:      "operation_duration_seconds",
			Help:      "The duration for a backup backupper operation.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"backupper", "operation", "success"}),

		k8sServiceOpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: promNamespace,
			Subsystem: promKubernetesSvcSubsystem,
			Name:      "operation_duration_seconds",
			Help:      "The duration for a kubernetes service operation.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"namespace", "operation", "success"}),
	}

	// Register metrics.
	reg.MustRegister(
		r.dexCliOpDuration,
		r.oidcProxyProvOpDuration,
		r.authBackAppRegOpDuration,
		r.backupBackupperOpDuration,
		r.k8sServiceOpDuration,
	)

	return r
}

func (r recorder) ObserveDexAuthBackendDexClientOp(_ context.Context, op string, success bool, startAt time.Time) {
	r.dexCliOpDuration.WithLabelValues(op, strconv.FormatBool(success)).
		Observe(time.Since(startAt).Seconds())
}

func (r recorder) ObserveOIDCProvisionerOperation(_ context.Context, provType, op string, success bool, startAt time.Time) {
	r.oidcProxyProvOpDuration.WithLabelValues(provType, op, strconv.FormatBool(success)).
		Observe(time.Since(startAt).Seconds())
}

func (r recorder) ObserveAuthBackendAppRegistererOperation(_ context.Context, appRegistererType, op string, success bool, startAt time.Time) {
	r.authBackAppRegOpDuration.WithLabelValues(appRegistererType, op, strconv.FormatBool(success)).
		Observe(time.Since(startAt).Seconds())
}

func (r recorder) ObserveBackupBackupperOperation(_ context.Context, backupperType, op string, success bool, startAt time.Time) {
	r.backupBackupperOpDuration.WithLabelValues(backupperType, op, strconv.FormatBool(success)).
		Observe(time.Since(startAt).Seconds())
}

func (r recorder) ObserveKubernetesServiceOperation(ctx context.Context, namespace, op string, success bool, startAt time.Time) {
	r.k8sServiceOpDuration.WithLabelValues(namespace, op, strconv.FormatBool(success)).
		Observe(time.Since(startAt).Seconds())
}

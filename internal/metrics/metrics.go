package metrics

import (
	"context"
	"time"

	koopercontroller "github.com/spotahome/kooper/controller"
)

// Recorder must be satisfied by all the backends that want to implement
// a metrics recorder.
type Recorder interface {
	koopercontroller.MetricsRecorder

	ObserveDexAuthBackendDexClientOp(ctx context.Context, op string, success bool, startAt time.Time)
	ObserveOIDCProvisionerOperation(ctx context.Context, provType, op string, success bool, startAt time.Time)
	ObserveAuthBackendAppRegistererOperation(ctx context.Context, appRegistererType, op string, success bool, startAt time.Time)
	ObserveBackupBackupperOperation(ctx context.Context, backupperType, op string, success bool, startAt time.Time)
	ObserveKubernetesServiceOperation(ctx context.Context, ns, op string, success bool, startAt time.Time)
}

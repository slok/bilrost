package backup

import (
	"context"
	"time"

	"github.com/slok/bilrost/internal/metrics"
	"github.com/slok/bilrost/internal/model"
)

type measuredBackupper struct {
	backupperType string
	rec           metrics.Recorder
	next          Backupper
}

// NewMeasuredbackupper returns a measured Backupper using a metrics.Recorder.
func NewMeasuredbackupper(backupperType string, rec metrics.Recorder, next Backupper) Backupper {
	return measuredBackupper{backupperType: backupperType, rec: rec, next: next}
}

func (m measuredBackupper) BackupOrGet(ctx context.Context, app model.App, data Data) (d *Data, err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveBackupBackupperOperation(ctx, m.backupperType, "BackupOrGet", err == nil, t0)
	}(time.Now())
	return m.next.BackupOrGet(ctx, app, data)
}

func (m measuredBackupper) GetBackup(ctx context.Context, app model.App) (d *Data, err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveBackupBackupperOperation(ctx, m.backupperType, "GetBackup", err == nil, t0)
	}(time.Now())
	return m.next.GetBackup(ctx, app)
}

func (m measuredBackupper) DeleteBackup(ctx context.Context, app model.App) (err error) {
	defer func(t0 time.Time) {
		m.rec.ObserveBackupBackupperOperation(ctx, m.backupperType, "DeleteBackup", err == nil, t0)
	}(time.Now())
	return m.next.DeleteBackup(ctx, app)
}

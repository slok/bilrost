package backup

import (
	"context"

	"github.com/slok/bilrost/internal/model"
)

// Data is the data that needs to be backuped.
type Data struct {
	AuthBackendID         string `json:"authBackendID"`
	ServiceName           string `json:"serviceName"`
	ServicePortOrNamePort string `json:"servicePortOrNamePort"`
}

// Backupper knows how to backup information to undo the security process.
type Backupper interface {
	// BackupOrGet will backup if the backup is not yet stored, otherwise
	// it will not backup and get the current backup data instead.
	BackupOrGet(ctx context.Context, app model.App, data Data) (*Data, error)
	// GetBackup gets an app Backup data.
	GetBackup(ctx context.Context, app model.App) (*Data, error)
	// DeleteBackup gets the backup.
	DeleteBackup(ctx context.Context, app model.App) error
}

//go:generate mockery -case underscore -output backupmock -outpkg backupmock -name Backupper

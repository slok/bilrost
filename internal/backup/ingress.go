package backup

import (
	"context"
	"encoding/json"
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/model"
)

const (
	ingressBackupAnnotation = "auth.bilrost.slok.dev/backup"
)

// KubernetesRepository is the proxy kubernetes service used to communicate with Kubernetes.
type KubernetesRepository interface {
	GetIngress(ctx context.Context, ns, name string) (*networkingv1.Ingress, error)
	UpdateIngress(ctx context.Context, ingress *networkingv1.Ingress) error
}

//go:generate mockery -case underscore -output backupmock -outpkg backupmock -name KubernetesRepository

type ingressBackupper struct {
	kuberepo KubernetesRepository
	logger   log.Logger
}

// NewIngressBackupper returns a new backuper that will make the backups on the application
// ingress kubernetes resource.
func NewIngressBackupper(kuberepo KubernetesRepository, logger log.Logger) Backupper {
	return ingressBackupper{
		kuberepo: kuberepo,
		logger:   logger.WithKV(log.KV{"service": "backup.IngressBackupper"}),
	}
}

func (i ingressBackupper) BackupOrGet(ctx context.Context, app model.App, data Data) (*Data, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("could not marshall data for backup: %w", err)
	}

	ing, err := i.kuberepo.GetIngress(ctx, app.Ingress.Namespace, app.Ingress.Name)
	if err != nil {
		return nil, fmt.Errorf("could not get ingress for backup: %w", err)
	}

	if ing.Annotations == nil {
		ing.Annotations = map[string]string{}
	}

	// If backup already stored return the backup.
	storedData, ok := ing.Annotations[ingressBackupAnnotation]
	if ok {
		data := &Data{}
		err := json.Unmarshal([]byte(storedData), data)
		if err != nil {
			return nil, fmt.Errorf("could not unmarshall from JSON stored backup: %w", err)
		}

		return data, nil
	}

	// Store backup.
	ing.Annotations[ingressBackupAnnotation] = string(jsonData)
	err = i.kuberepo.UpdateIngress(ctx, ing)
	if err != nil {
		return nil, fmt.Errorf("could not update ingress for backup: %w", err)
	}

	return &data, nil
}

func (i ingressBackupper) GetBackup(ctx context.Context, app model.App) (*Data, error) {
	ing, err := i.kuberepo.GetIngress(ctx, app.Ingress.Namespace, app.Ingress.Name)
	if err != nil {
		return nil, fmt.Errorf("could not get ingress for backup: %w", err)
	}

	if ing.Annotations == nil {
		return nil, fmt.Errorf("backup not present")
	}

	storedData, ok := ing.Annotations[ingressBackupAnnotation]
	if !ok {
		return nil, fmt.Errorf("backup not present")
	}

	data := &Data{}
	err = json.Unmarshal([]byte(storedData), data)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshall from JSON stored backup: %w", err)
	}

	return data, nil
}

func (i ingressBackupper) DeleteBackup(ctx context.Context, app model.App) error {
	ing, err := i.kuberepo.GetIngress(ctx, app.Ingress.Namespace, app.Ingress.Name)
	if err != nil {
		return fmt.Errorf("could not get ingress for backup: %w", err)
	}

	if ing.Annotations == nil {
		ing.Annotations = map[string]string{}
	}

	_, ok := ing.Annotations[ingressBackupAnnotation]
	if !ok {
		return nil
	}

	delete(ing.Annotations, ingressBackupAnnotation)

	err = i.kuberepo.UpdateIngress(ctx, ing)
	if err != nil {
		return fmt.Errorf("could not update ingress for backup: %w", err)
	}

	return nil
}

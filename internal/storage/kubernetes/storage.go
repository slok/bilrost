package kubernetes

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/bilrost/internal/model"
	"github.com/slok/bilrost/internal/security"
	authv1 "github.com/slok/bilrost/pkg/apis/auth/v1"
	kubernetesbilrost "github.com/slok/bilrost/pkg/kubernetes/gen/clientset/versioned"
)

// Repository knows how to deal with CRUD operations on a storage.
type Repository struct {
	cli kubernetesbilrost.Interface
}

// NewRepository returns a new repository
func NewRepository(cli kubernetesbilrost.Interface) Repository {
	return Repository{
		cli: cli,
	}
}

// GetAuthBackend satisifies controller.AuthBackendRepository interface.
func (r Repository) GetAuthBackend(ctx context.Context, id string) (*model.AuthBackend, error) {
	ab, err := r.cli.AuthV1().AuthBackends().Get(id, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not retrieve auth backend from Kubernetes: %w", err)
	}

	res := mapAuthBackendK8sToModel(ab)

	return res, nil
}

func mapAuthBackendK8sToModel(ab *authv1.AuthBackend) *model.AuthBackend {
	res := &model.AuthBackend{ID: ab.Name}

	switch {
	case ab.Spec.Dex != nil:
		res.Dex = &model.AuthBackendDex{
			APIURL:    ab.Spec.Dex.APIAddress,
			PublicURL: ab.Spec.Dex.PublicURL,
		}
	}

	return res
}

// Interface implementation checks.
var _ security.AuthBackendRepository = Repository{}

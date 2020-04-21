package backup_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/slok/bilrost/internal/backup"
	"github.com/slok/bilrost/internal/backup/backupmock"
	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/model"
)

func TestIngressBackupperBackupOrGet(t *testing.T) {
	tests := map[string]struct {
		app     model.App
		data    backup.Data
		mock    func(m *backupmock.KubernetesRepository)
		expData backup.Data
		expErr  bool
	}{
		"If the data does not exists, the data should be stored on the ingress.": {
			app: model.App{
				Ingress: model.KubernetesIngress{
					Name:      "test-ing",
					Namespace: "test-ns",
				},
			},
			data: backup.Data{
				AuthBackendID:         "auth-test",
				ServiceName:           "test-svc",
				ServicePortOrNamePort: "http",
			},
			mock: func(m *backupmock.KubernetesRepository) {
				ing := &networkingv1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ing",
						Annotations: map[string]string{
							"test": "test1",
						},
					},
				}
				m.On("GetIngress", mock.Anything, "test-ns", "test-ing").Once().Return(ing, nil)

				expIng := ing.DeepCopy()
				expIng.Annotations["auth.bilrost.slok.dev/backup"] = `{"authBackendID":"auth-test","serviceName":"test-svc","servicePortOrNamePort":"http"}`
				m.On("UpdateIngress", mock.Anything, expIng).Once().Return(nil)
			},
			expData: backup.Data{
				AuthBackendID:         "auth-test",
				ServiceName:           "test-svc",
				ServicePortOrNamePort: "http",
			},
		},

		"If the data already exists, it should not store and return the already stored data.": {
			app: model.App{
				Ingress: model.KubernetesIngress{
					Name:      "test-ing",
					Namespace: "test-ns",
				},
			},
			data: backup.Data{
				ServiceName:           "test-svc",
				ServicePortOrNamePort: "http",
			},
			mock: func(m *backupmock.KubernetesRepository) {
				ing := &networkingv1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ing",
						Annotations: map[string]string{
							"test":                         "test1",
							"auth.bilrost.slok.dev/backup": `{"authBackendID":"auth-test2","serviceName":"test-svc2","servicePortOrNamePort":"8080"}`,
						},
					},
				}
				m.On("GetIngress", mock.Anything, "test-ns", "test-ing").Once().Return(ing, nil)
			},
			expData: backup.Data{
				AuthBackendID:         "auth-test2",
				ServiceName:           "test-svc2",
				ServicePortOrNamePort: "8080",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			mk := &backupmock.KubernetesRepository{}
			test.mock(mk)

			bk := backup.NewIngressBackupper(mk, log.Dummy)
			data, err := bk.BackupOrGet(context.TODO(), test.app, test.data)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				mk.AssertExpectations(t)
				assert.Equal(&test.expData, data)
			}
		})
	}
}

func TestIngressBackupperGetBackup(t *testing.T) {
	tests := map[string]struct {
		app     model.App
		mock    func(m *backupmock.KubernetesRepository)
		expData backup.Data
		expErr  bool
	}{
		"If the data does not exists, it should return an error.": {
			app: model.App{
				Ingress: model.KubernetesIngress{
					Name:      "test-ing",
					Namespace: "test-ns",
				},
			},
			mock: func(m *backupmock.KubernetesRepository) {
				ing := &networkingv1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ing",
						Annotations: map[string]string{
							"test": "test1",
						},
					},
				}
				m.On("GetIngress", mock.Anything, "test-ns", "test-ing").Once().Return(ing, nil)
			},
			expErr: true,
		},
		"If the data exists, it should return the data.": {
			app: model.App{
				Ingress: model.KubernetesIngress{
					Name:      "test-ing",
					Namespace: "test-ns",
				},
			},
			mock: func(m *backupmock.KubernetesRepository) {
				ing := &networkingv1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ing",
						Annotations: map[string]string{
							"test":                         "test1",
							"auth.bilrost.slok.dev/backup": `{"authBackendID":"auth-test","serviceName":"test-svc2","servicePortOrNamePort":"8080"}`,
						},
					},
				}
				m.On("GetIngress", mock.Anything, "test-ns", "test-ing").Once().Return(ing, nil)
			},
			expData: backup.Data{
				AuthBackendID:         "auth-test",
				ServiceName:           "test-svc2",
				ServicePortOrNamePort: "8080",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			mk := &backupmock.KubernetesRepository{}
			test.mock(mk)

			bk := backup.NewIngressBackupper(mk, log.Dummy)
			data, err := bk.GetBackup(context.TODO(), test.app)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				mk.AssertExpectations(t)
				assert.Equal(&test.expData, data)
			}
		})
	}
}

func TestIngressBackupperDeleteBackup(t *testing.T) {
	tests := map[string]struct {
		app    model.App
		mock   func(m *backupmock.KubernetesRepository)
		expErr bool
	}{
		"If the backup data is already stored it should delete it from the ingress.": {
			app: model.App{
				Ingress: model.KubernetesIngress{
					Name:      "test-ing",
					Namespace: "test-ns",
				},
			},
			mock: func(m *backupmock.KubernetesRepository) {
				ing := &networkingv1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ing",
						Annotations: map[string]string{
							"test":                         "test1",
							"auth.bilrost.slok.dev/backup": `{"authBackendID":"auth-test","serviceName":"test-svc","servicePortOrNamePort":"http"}`,
						},
					},
				}
				m.On("GetIngress", mock.Anything, "test-ns", "test-ing").Once().Return(ing, nil)

				expIng := ing.DeepCopy()
				delete(expIng.Annotations, "auth.bilrost.slok.dev/backup")
				m.On("UpdateIngress", mock.Anything, expIng).Once().Return(nil)
			},
		},

		"If the backup data is not stored, it should not delete the data from the ingress.": {
			app: model.App{
				Ingress: model.KubernetesIngress{
					Name:      "test-ing",
					Namespace: "test-ns",
				},
			},
			mock: func(m *backupmock.KubernetesRepository) {
				ing := &networkingv1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ing",
						Annotations: map[string]string{
							"test": "test1",
						},
					},
				}
				m.On("GetIngress", mock.Anything, "test-ns", "test-ing").Once().Return(ing, nil)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			mk := &backupmock.KubernetesRepository{}
			test.mock(mk)

			bk := backup.NewIngressBackupper(mk, log.Dummy)
			err := bk.DeleteBackup(context.TODO(), test.app)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				mk.AssertExpectations(t)
			}
		})
	}
}

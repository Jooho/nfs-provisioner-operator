package resources

import (
	"context"

	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/controllers/defaults"
)

// StorageClassManager manages StorageClass resources
type StorageClassManager struct {
	BaseResourceManager
}

// NewStorageClassManager creates a new StorageClassManager
func NewStorageClassManager(base BaseResourceManager) *StorageClassManager {
	return &StorageClassManager{
		BaseResourceManager: base,
	}
}

// GetResourceName returns the name of the resource this manager handles
func (m *StorageClassManager) GetResourceName() string {
	return "StorageClass"
}

// EnsureResource ensures the StorageClass exists
func (m *StorageClassManager) EnsureResource(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error {
	log := m.Log.WithValues("resource", m.GetResourceName())

	// Check if the storageclass already exists
	scName := defaults.SCForNFSProvisioner

	if nfsProvisioner.Spec.SCForNFSProvisioner != "" {
		scName = nfsProvisioner.Spec.SCForNFSProvisioner
	}

	scFound := &storagev1.StorageClass{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: scName, Namespace: ""}, scFound)
	if err != nil && errors.IsNotFound(err) {
		// Define a new storageclass
		sc := m.buildStorageClass(nfsProvisioner)

		log.Info("Creating a new Storageclass", "Storageclass.Name", sc.Name)

		if err = m.Client.Create(ctx, sc); err != nil {
			log.Error(err, "Failed to create a Storageclass for NFSProvisioner", "Storageclass.Name", sc.Name)
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

// buildStorageClass creates a new StorageClass object
func (m *StorageClassManager) buildStorageClass(nfsProvisioner *cachev1alpha1.NFSProvisioner) *storagev1.StorageClass {
	scName := defaults.SCForNFSProvisioner

	if nfsProvisioner.Spec.SCForNFSProvisioner != "" {
		scName = nfsProvisioner.Spec.SCForNFSProvisioner
	}
	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: scName,
		},
		Provisioner: "example.com/nfs",
		Parameters:  map[string]string{"mountOptions": "vers=4.1"},
	}

	ctrl.SetControllerReference(nfsProvisioner, sc, m.Scheme)
	return sc
}

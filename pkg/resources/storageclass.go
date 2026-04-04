package resources

import (
	"context"

	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/builder"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
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
	log := m.Log.WithValues(
		"resource", m.GetResourceName(),
		"nfsprovisioner.name", nfsProvisioner.Name,
		"nfsprovisioner.namespace", nfsProvisioner.Namespace,
	)

	// Determine StorageClass name
	scName := defaults.SCForNFSProvisioner
	if nfsProvisioner.Spec.SCForNFSProvisioner != "" {
		scName = nfsProvisioner.Spec.SCForNFSProvisioner
	}

	// Check if the storageclass already exists
	scFound := &storagev1.StorageClass{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: scName, Namespace: ""}, scFound)
	if err != nil && errors.IsNotFound(err) {
		sc := builder.BuildStorageClass(nfsProvisioner)
		log.Info("Creating StorageClass", "storageclass.name", sc.Name)
		if createErr := m.Client.Create(ctx, sc); createErr != nil {
			log.Error(createErr, "Failed to create StorageClass", "storageclass.name", sc.Name)
			return createErr
		}
		log.Info("Successfully created StorageClass", "storageclass.name", sc.Name)
	} else if err != nil {
		log.Error(err, "Failed to get StorageClass")
		return err
	} else {
		log.V(1).Info("StorageClass already exists", "storageclass.name", scFound.Name)
	}

	return nil
}

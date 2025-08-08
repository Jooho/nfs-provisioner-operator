package resources

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
)

// ResourceManagerSet holds all resource managers
type ResourceManagerSet struct {
	// Phase 2 resources
	SCC            ResourceManager
	PVC            ResourceManager
	ServiceAccount ResourceManager
	// Phase 3 resources
	RBAC         ResourceManager
	Deployment   ResourceManager
	Service      ResourceManager
	StorageClass ResourceManager
}

// NewResourceManagerSet creates a new set of resource managers
func NewResourceManagerSet(client client.Client, log logr.Logger, scheme *runtime.Scheme) *ResourceManagerSet {
	base := NewBaseResourceManager(client, log, scheme)

	return &ResourceManagerSet{
		// Phase 2 resources
		SCC:            NewSCCManager(base),
		PVC:            NewPVCManager(base),
		ServiceAccount: NewServiceAccountManager(base),
		// Phase 3 resources
		RBAC:         NewRBACManager(base),
		Deployment:   NewDeploymentManager(base),
		Service:      NewServiceManager(base),
		StorageClass: NewStorageClassManager(base),
	}
}

// EnsureAllResources ensures all managed resources exist in the correct state
func (r *ResourceManagerSet) EnsureAllResources(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error {
	// List of managers to process in order
	managers := []ResourceManager{
		// Phase 2 resources
		r.SCC,
		r.PVC,
		r.ServiceAccount,
		// Phase 3 resources
		r.RBAC,
		r.Deployment,
		r.Service,
		r.StorageClass,
	}

	// Process each manager
	for _, manager := range managers {
		if err := manager.EnsureResource(ctx, nfsProvisioner); err != nil {
			return err
		}
	}

	return nil
}

// GetManagedResourceNames returns the names of all resources managed by this set
func (r *ResourceManagerSet) GetManagedResourceNames() []string {
	return []string{
		r.SCC.GetResourceName(),
		r.PVC.GetResourceName(),
		r.ServiceAccount.GetResourceName(),
		r.RBAC.GetResourceName(),
		r.Deployment.GetResourceName(),
		r.Service.GetResourceName(),
		r.StorageClass.GetResourceName(),
	}
}

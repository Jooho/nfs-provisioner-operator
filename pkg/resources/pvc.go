package resources

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/builder"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
)

// PVCManager manages PersistentVolumeClaim resources
type PVCManager struct {
	BaseResourceManager
}

// NewPVCManager creates a new PVCManager
func NewPVCManager(base BaseResourceManager) *PVCManager {
	return &PVCManager{
		BaseResourceManager: base,
	}
}

// GetResourceName returns the name of the resource this manager handles
func (m *PVCManager) GetResourceName() string {
	return "PersistentVolumeClaim"
}

// EnsureResource ensures the PVC exists when using PVC storage type
func (m *PVCManager) EnsureResource(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error {
	log := m.Log.WithValues(
		"resource", m.GetResourceName(),
		"nfsprovisioner.name", nfsProvisioner.Name,
		"nfsprovisioner.namespace", nfsProvisioner.Namespace,
	)

	// Build PVC using builder (returns nil if not needed)
	pvc := builder.BuildPVC(nfsProvisioner)
	if pvc == nil {
		log.V(1).Info("Skipping PVC creation - not using PVC storage")
		return nil
	}

	// Determine PVC name (use existing PVC if specified)
	pvcName := defaults.Pvc
	if nfsProvisioner.Spec.Pvc != "" {
		pvcName = nfsProvisioner.Spec.Pvc
	}

	// Check if PVC already exists
	pvcFound := &corev1.PersistentVolumeClaim{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: nfsProvisioner.Namespace}, pvcFound)
	if err != nil && errors.IsNotFound(err) {
		// Only create PVC if we're supposed to manage it (not using existing PVC)
		if nfsProvisioner.Spec.Pvc != "" {
			// User specified an existing PVC that doesn't exist
			log.Error(err, "Specified PVC does not exist", "pvc.name", pvcName)
			return err
		}

		// Set NFSProvisioner instance as the owner and controller
		if err := ctrl.SetControllerReference(nfsProvisioner, pvc, m.Scheme); err != nil {
			log.Error(err, "Failed to set controller reference on PVC")
			return err
		}

		log.Info("Creating PersistentVolumeClaim", "pvc.namespace", pvc.Namespace, "pvc.name", pvc.Name)
		if err = m.Client.Create(ctx, pvc); err != nil {
			log.Error(err, "Failed to create PersistentVolumeClaim", "pvc.namespace", pvc.Namespace, "pvc.name", pvc.Name)
			return err
		}
		log.Info("Successfully created PersistentVolumeClaim", "pvc.namespace", pvc.Namespace, "pvc.name", pvc.Name)
	} else if err != nil {
		log.Error(err, "Failed to get PersistentVolumeClaim")
		return err
	} else {
		log.V(1).Info("PersistentVolumeClaim already exists", "pvc.namespace", pvcFound.Namespace, "pvc.name", pvcFound.Name)
	}

	return nil
}

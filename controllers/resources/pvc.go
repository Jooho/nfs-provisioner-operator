package resources

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/controllers/defaults"
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
	log := m.Log.WithValues("resource", m.GetResourceName())

	// Skip PVC creation if using hostPath
	if nfsProvisioner.Spec.HostPathDir != "" {
		log.Info("Skipping PVC creation - using hostPath storage")
		return nil
	}

	// Determine PVC name
	pvcName := defaults.Pvc
	if nfsProvisioner.Spec.Pvc != "" {
		pvcName = nfsProvisioner.Spec.Pvc
	}

	// Check if PVC already exists
	pvcFound := &corev1.PersistentVolumeClaim{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: nfsProvisioner.Namespace}, pvcFound)

	if err != nil {
		if errors.IsNotFound(err) {
			// Only create PVC if we're supposed to manage it (not using existing PVC)
			if nfsProvisioner.Spec.Pvc == "" {
				pvc := m.buildPVC(nfsProvisioner)
				log.Info("Creating a new PersistentVolumeClaim", "PersistentVolumeClaim.Namespace", pvc.Namespace, "PersistentVolumeClaim.Name", pvc.Name)

				if err := m.Client.Create(ctx, pvc); err != nil {
					log.Error(err, "Failed to create a new PersistentVolumeClaim", "PersistentVolumeClaim.Namespace", pvc.Namespace, "PersistentVolumeClaim.Name", pvc.Name)
					return err
				}
			} else {
				// User specified an existing PVC that doesn't exist
				log.Error(err, "Specified PVC does not exist", "PVC.Name", pvcName)
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

// buildPVC creates a new PersistentVolumeClaim object
func (m *PVCManager) buildPVC(nfsProvisioner *cachev1alpha1.NFSProvisioner) *corev1.PersistentVolumeClaim {
	// Determine storage class name
	scName := defaults.SCForNFSPvc
	if nfsProvisioner.Spec.SCForNFSPvc != "" {
		scName = nfsProvisioner.Spec.SCForNFSPvc
	}

	// Determine storage size
	pvcSize := defaults.StorageSize
	if nfsProvisioner.Spec.StorageSize != "" {
		pvcSize = nfsProvisioner.Spec.StorageSize
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaults.Pvc,
			Namespace: nfsProvisioner.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(pvcSize),
				},
			},
			StorageClassName: &scName,
		},
	}

	// Set controller reference
	ctrl.SetControllerReference(nfsProvisioner, pvc, m.Scheme)
	return pvc
}

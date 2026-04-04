package defaults

import (
	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

// Default values for NFSProvisioner resources
const (
	// SecurityContextConstraints is the permission control mechanism in OpenShift
	SecurityContextConstraints = "nfs-provisioner"

	// HostPathDir is the default directory that NFS server will use
	HostPathDir = "/home/core/nfs"

	// StorageSize is the default PVC size for NFS server
	StorageSize = "10Gi"

	// Pvc is the default storage for NFS server
	Pvc = "nfs-server"

	// SCForNFSPvc is the default storageClass name to create PVC for NFS server
	SCForNFSPvc = "local-sc"

	// ServiceAccount is the project level main sa that has power to control NFS provisioners
	ServiceAccount = "nfs-provisioner"

	// ClusterRole is for NFS Provisioner to create SC/PV/PVC
	ClusterRole = "nfs-provisioner-runner"

	// ClusterRoleBinding matches ClusterRole and ServiceAccount
	ClusterRoleBinding = "nfs-provisioner-runner"

	// Role gives the permissions to get endpoints/services for NFS server
	Role = "leader-locking-nfs-provisioner"

	// RoleBinding gives the Role to the SA
	RoleBinding = "leader-locking-nfs-provisioner"

	// Deployment is for NFS server
	Deployment = "nfs-provisioner"

	// Service is for NFS provisioner to access to NFS Server
	Service = "nfs-provisioner"

	// SCForNFSProvisioner is for NFS Provisioner
	SCForNFSProvisioner = "nfs"

	// NFSImage is the default NFS provisioner image
	NFSImage = "registry.k8s.io/sig-storage/nfs-provisioner:v4.0.8"

	// NFSImagePullPolicy is the default pull policy for NFS provisioner image
	NFSImagePullPolicy = corev1.PullIfNotPresent
)

// NodeSelector is for the node where NFS server will be running
var NodeSelector = map[string]string{"app": "nfs-provisioner"}

// ApplyDefaults mutates the NFSProvisioner resource to apply default values
// for any unset optional fields.
//
// Defaults:
//   - spec.storageSize: "10Gi"
//   - spec.scForNFS: "nfs"
//   - spec.nfsImageConfiguration.image: (default NFS provisioner image)
//   - spec.nfsImageConfiguration.imagePullPolicy: "IfNotPresent"
func ApplyDefaults(nfs *cachev1alpha1.NFSProvisioner) {
	// Apply default storageSize if not set
	if nfs.Spec.StorageSize == "" {
		nfs.Spec.StorageSize = StorageSize
	}

	// Apply default scForNFS if not set
	if nfs.Spec.SCForNFSProvisioner == "" {
		nfs.Spec.SCForNFSProvisioner = SCForNFSProvisioner
	}

	// Apply defaults for NFSImageConfiguration
	if nfs.Spec.NFSImageConfiguration == nil {
		nfs.Spec.NFSImageConfiguration = &cachev1alpha1.ImageConfiguration{}
	}

	// Apply default NFS image if not set
	if nfs.Spec.NFSImageConfiguration.Image == nil || *nfs.Spec.NFSImageConfiguration.Image == "" {
		nfs.Spec.NFSImageConfiguration.Image = ptr.To(NFSImage)
	}

	// Apply default image pull policy if not set
	if nfs.Spec.NFSImageConfiguration.ImagePullPolicy == nil {
		policy := corev1.PullIfNotPresent
		nfs.Spec.NFSImageConfiguration.ImagePullPolicy = &policy
	}
}

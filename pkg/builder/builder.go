package builder

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
)

// Resource-specific builder functions for NFSProvisioner resources.
// These are stateless, pure functions that transform NFSProvisioner spec
// into Kubernetes resource specifications.

// BuildDeployment constructs a Deployment resource for the NFS server.
func BuildDeployment(nfs *cachev1alpha1.NFSProvisioner) *appsv1.Deployment {
	ls := labelsForNFSProvisioner(nfs.Name)

	// Determine storage type
	storageType := determineStorageType(nfs)

	// Apply image configuration
	nfsImage := defaults.NFSImage
	nfsImagePullPolicy := defaults.NFSImagePullPolicy
	if nfs.Spec.NFSImageConfiguration != nil {
		if nfs.Spec.NFSImageConfiguration.Image != nil {
			nfsImage = *nfs.Spec.NFSImageConfiguration.Image
		}
		if nfs.Spec.NFSImageConfiguration.ImagePullPolicy != nil {
			nfsImagePullPolicy = *nfs.Spec.NFSImageConfiguration.ImagePullPolicy
		}
	}

	// Apply node selector
	nodeSelector := defaults.NodeSelector
	if nfs.Spec.NodeSelector != nil {
		nodeSelector = nfs.Spec.NodeSelector
	}
	// PVC-based storage doesn't need node selector
	if storageType == "PVC" {
		nodeSelector = map[string]string{}
	}

	// Get volume specification
	volumeSource := getVolumeSpec(nfs, storageType)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaults.Deployment,
			Namespace: nfs.Namespace,
			Labels:    ls,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           nfsImage,
						ImagePullPolicy: nfsImagePullPolicy,
						Name:            "nfs-provisioner",
						Ports:           buildNFSPorts(),
						SecurityContext: &corev1.SecurityContext{
							Capabilities: &corev1.Capabilities{
								Add:  []corev1.Capability{"DAC_READ_SEARCH", "SYS_RESOURCE"},
								Drop: []corev1.Capability{"KILL", "MKNOD", "SYS_CHROOT"},
							},
						},
						Args: []string{"'-provisioner=example.com/nfs'"},
						Env:  buildNFSEnv(),
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "export-volume",
							MountPath: "/export",
						}},
					}},
					NodeSelector:       nodeSelector,
					ServiceAccountName: defaults.ServiceAccount,
					Volumes: []corev1.Volume{{
						Name:         "export-volume",
						VolumeSource: *volumeSource,
					}},
				},
			},
		},
	}

	return dep
}

// BuildService constructs a Service resource exposing the NFS server.
func BuildService(nfs *cachev1alpha1.NFSProvisioner) *corev1.Service {
	ls := labelsForNFSProvisioner(nfs.Name)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaults.Service,
			Namespace: nfs.Namespace,
			Labels:    ls,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Ports:    buildServicePorts(),
		},
	}

	return svc
}

// BuildServiceAccount constructs a ServiceAccount for the NFS provisioner pod.
func BuildServiceAccount(nfs *cachev1alpha1.NFSProvisioner) *corev1.ServiceAccount {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaults.ServiceAccount,
			Namespace: nfs.Namespace,
		},
	}

	return sa
}

// BuildPVC constructs a PVC for NFS server storage (if spec.scForNFSPvc is set).
// Returns nil if PVC should not be created.
func BuildPVC(nfs *cachev1alpha1.NFSProvisioner) *corev1.PersistentVolumeClaim {
	if nfs.Spec.SCForNFSPvc == "" {
		return nil
	}

	storageSize := nfs.Spec.StorageSize
	if storageSize == "" {
		storageSize = defaults.StorageSize
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaults.Pvc,
			Namespace: nfs.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &nfs.Spec.SCForNFSPvc,
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: parseQuantity(storageSize),
				},
			},
		},
	}

	return pvc
}

// BuildStorageClass constructs a StorageClass for end users.
func BuildStorageClass(nfs *cachev1alpha1.NFSProvisioner) *storagev1.StorageClass {
	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: nfs.Spec.SCForNFSProvisioner,
		},
		Provisioner: "example.com/nfs",
		Parameters: map[string]string{
			"archiveOnDelete": "false",
		},
	}

	return sc
}

// BuildClusterRole constructs a ClusterRole for the NFS provisioner.
func BuildClusterRole() *rbacv1.ClusterRole {
	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaults.ClusterRole,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumes"},
				Verbs:     []string{"get", "list", "watch", "create", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumeclaims"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"storageclasses"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"create", "update", "patch"},
			},
		},
	}

	return cr
}

// BuildClusterRoleBinding constructs a ClusterRoleBinding for the NFS provisioner.
func BuildClusterRoleBinding(nfs *cachev1alpha1.NFSProvisioner) *rbacv1.ClusterRoleBinding {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaults.ClusterRoleBinding,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      defaults.ServiceAccount,
			Namespace: nfs.Namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     defaults.ClusterRole,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	return crb
}

// Helper functions

// labelsForNFSProvisioner returns the labels for selecting the resources belonging to the NFSProvisioner CR
func labelsForNFSProvisioner(name string) map[string]string {
	return map[string]string{"app": "nfs-provisioner", "nfsprovisioner_cr": name}
}

// determineStorageType returns the storage type based on NFSProvisioner spec
func determineStorageType(nfs *cachev1alpha1.NFSProvisioner) string {
	if nfs.Spec.Pvc != "" {
		return "PVC"
	} else if nfs.Spec.SCForNFSPvc != "" {
		return "PVC"
	}
	return "HostPath"
}

// getVolumeSpec returns the appropriate volume source based on storage type
func getVolumeSpec(nfs *cachev1alpha1.NFSProvisioner, storageType string) *corev1.VolumeSource {
	if storageType == "PVC" {
		pvcName := nfs.Spec.Pvc
		if pvcName == "" {
			pvcName = defaults.Pvc
		}
		return &corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvcName,
			},
		}
	}

	hostPathDir := nfs.Spec.HostPathDir
	if hostPathDir == "" {
		hostPathDir = defaults.HostPathDir
	}
	hostPathType := corev1.HostPathDirectory
	return &corev1.VolumeSource{
		HostPath: &corev1.HostPathVolumeSource{
			Path: hostPathDir,
			Type: &hostPathType,
		},
	}
}

// buildNFSPorts returns the port configuration for NFS container
func buildNFSPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{Name: "nfs", ContainerPort: 2049},
		{Name: "nfs-udp", ContainerPort: 2049, Protocol: "UDP"},
		{Name: "nlockmgr", ContainerPort: 32803},
		{Name: "nlockmgr-udp", ContainerPort: 32803, Protocol: "UDP"},
		{Name: "mountd", ContainerPort: 20048},
		{Name: "mountd-udp", ContainerPort: 20048, Protocol: "UDP"},
		{Name: "rquotad", ContainerPort: 875},
		{Name: "rquotad-udp", ContainerPort: 875, Protocol: "UDP"},
		{Name: "rpcbind", ContainerPort: 111},
		{Name: "rpcbind-udp", ContainerPort: 111, Protocol: "UDP"},
		{Name: "statd", ContainerPort: 662},
		{Name: "statd-udp", ContainerPort: 662, Protocol: "UDP"},
	}
}

// buildNFSEnv returns the environment variables for NFS container
func buildNFSEnv() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
		{
			Name:  "SERVICE_NAME",
			Value: "nfs-provisioner",
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	}
}

// buildServicePorts returns the port configuration for NFS service
func buildServicePorts() []corev1.ServicePort {
	return []corev1.ServicePort{
		{Name: "nfs", Port: 2049},
		{Name: "nfs-udp", Port: 2049, Protocol: "UDP"},
		{Name: "nlockmgr", Port: 32803},
		{Name: "nlockmgr-udp", Port: 32803, Protocol: "UDP"},
		{Name: "mountd", Port: 20048},
		{Name: "mountd-udp", Port: 20048, Protocol: "UDP"},
		{Name: "rquotad", Port: 875},
		{Name: "rquotad-udp", Port: 875, Protocol: "UDP"},
		{Name: "rpcbind", Port: 111},
		{Name: "rpcbind-udp", Port: 111, Protocol: "UDP"},
		{Name: "statd", Port: 662},
		{Name: "statd-udp", Port: 662, Protocol: "UDP"},
	}
}

// parseQuantity parses a string quantity into resource.Quantity
func parseQuantity(s string) resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		// Return default if parsing fails
		return resource.MustParse(defaults.StorageSize)
	}
	return q
}

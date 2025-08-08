package resources

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/controllers/defaults"
)

// DeploymentManager manages Deployment resources
type DeploymentManager struct {
	BaseResourceManager
}

// NewDeploymentManager creates a new DeploymentManager
func NewDeploymentManager(base BaseResourceManager) *DeploymentManager {
	return &DeploymentManager{
		BaseResourceManager: base,
	}
}

// GetResourceName returns the name of the resource this manager handles
func (m *DeploymentManager) GetResourceName() string {
	return "Deployment"
}

// EnsureResource ensures the Deployment exists
func (m *DeploymentManager) EnsureResource(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error {
	log := m.Log.WithValues("resource", m.GetResourceName())

	// Determine storage type for deployment
	storageType := "PVC"
	if nfsProvisioner.Spec.HostPathDir != "" {
		storageType = "HOSTPATH"
	}

	// Check if the deployment already exists
	deployFound := &appsv1.Deployment{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.Deployment, Namespace: nfsProvisioner.Namespace}, deployFound)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := m.buildDeployment(nfsProvisioner, storageType)

		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		if err = m.Client.Create(ctx, dep); err != nil {
			log.Error(err, "Failed to create a Deployment for NFSProvisioner", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

// buildDeployment creates a new Deployment object
func (m *DeploymentManager) buildDeployment(nfsProvisioner *cachev1alpha1.NFSProvisioner, storageType string) *appsv1.Deployment {
	ls := labelsForNFSProvisioner(nfsProvisioner.Name)

	nodeSelector := defaults.NodeSelector
	nfsImage := defaults.NFSImage
	nfsImagePullPolicy := defaults.NFSImagePullPolicy

	if nfsProvisioner.Spec.NodeSelector != nil {
		nodeSelector = nfsProvisioner.Spec.NodeSelector
	}

	if nfsProvisioner.Spec.NFSImageConfiguration != nil {
		if nfsProvisioner.Spec.NFSImageConfiguration.Image != nil {
			nfsImage = *nfsProvisioner.Spec.NFSImageConfiguration.Image
		}

		if nfsProvisioner.Spec.NFSImageConfiguration.ImagePullPolicy != nil {
			nfsImagePullPolicy = *nfsProvisioner.Spec.NFSImageConfiguration.ImagePullPolicy
		}
	}

	if storageType == "PVC" {
		nodeSelector = map[string]string{}
	}

	sa := defaults.ServiceAccount

	volumeSourceSpec := m.getVolumeSpec(nfsProvisioner, storageType)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaults.Deployment,
			Namespace: nfsProvisioner.Namespace,
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
						Ports: []corev1.ContainerPort{
							{Name: "nfs",
								ContainerPort: 2049},
							{Name: "nfs-udp",
								ContainerPort: 2049,
								Protocol:      "UDP"},
							{Name: "nlockmgr",
								ContainerPort: 32803},
							{Name: "nlockmgr-udp",
								ContainerPort: 32803,
								Protocol:      "UDP"},
							{Name: "mountd",
								ContainerPort: 20048},
							{Name: "mountd-udp",
								ContainerPort: 20048,
								Protocol:      "UDP"},
							{Name: "rquotad",
								ContainerPort: 875},
							{Name: "rquotad-udp",
								ContainerPort: 875,
								Protocol:      "UDP"},
							{Name: "rpcbind",
								ContainerPort: 111},
							{Name: "rpcbind-udp",
								ContainerPort: 111,
								Protocol:      "UDP"},
							{Name: "statd",
								ContainerPort: 662},
							{Name: "statd-udp",
								ContainerPort: 662,
								Protocol:      "UDP"},
						},
						SecurityContext: &corev1.SecurityContext{
							Capabilities: &corev1.Capabilities{
								Add:  []corev1.Capability{"DAC_READ_SEARCH", "SYS_RESOURCE"},
								Drop: []corev1.Capability{"KILL", "MKNOD", "SYS_CHROOT"},
							},
						},

						Args: []string{"'-provisioner=example.com/nfs'"},
						Env: []corev1.EnvVar{{
							Name: "POD_IP",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "status.podIP",
								},
							},
						}, {
							Name:  "SERVICE_NAME",
							Value: "nfs-provisioner",
						}, {
							Name: "POD_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "export-volume",
							MountPath: "/export",
						}},
					}},
					NodeSelector:       nodeSelector,
					ServiceAccountName: sa,
					Volumes: []corev1.Volume{{
						Name:         "export-volume",
						VolumeSource: *volumeSourceSpec,
					}},
				},
			},
		},
	}

	// Set NFSProvisioner instance as the owner and controller
	ctrl.SetControllerReference(nfsProvisioner, dep, m.Scheme)
	return dep
}

// getVolumeSpec returns the appropriate volume source based on storage type
func (m *DeploymentManager) getVolumeSpec(nfsProvisioner *cachev1alpha1.NFSProvisioner, storageType string) *corev1.VolumeSource {
	log := m.Log.WithName("getVolumeSpec")
	hostPathDir := nfsProvisioner.Spec.HostPathDir

	pvcName := defaults.Pvc

	hostPathType := corev1.HostPathDirectory

	if nfsProvisioner.Spec.Pvc != "" {
		pvcName = nfsProvisioner.Spec.Pvc
	}

	if storageType == "HOSTPATH" {
		log.Info("StorageType is HOSTPATH")
		return &corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: hostPathDir,
				Type: &hostPathType,
			}}
	}

	log.Info("StorageType is PVC")
	return &corev1.VolumeSource{
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: pvcName,
		}}
}

// labelsForNFSProvisioner returns the labels for selecting the resources
// belonging to the given NFSProvisioner CR name.
func labelsForNFSProvisioner(name string) map[string]string {
	return map[string]string{"app": "nfs-provisioner", "nfsprovisioner_cr": name}
}

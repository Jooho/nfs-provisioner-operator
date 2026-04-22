/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NFSProvisionerSpec defines the desired state of NFSProvisioner
type NFSProvisionerSpec struct {
	NodeSelector          map[string]string   `json:"nodeSelector,omitempty"`
	NFSImageConfiguration *ImageConfiguration `json:"nfsImageConfiguration,omitempty"`
	HostPathDir           string              `json:"hostPathDir,omitempty"`
	Pvc                   string              `json:"pvc,omitempty"`
	StorageSize           string              `json:"storageSize,omitempty"`
	SCForNFSPvc           string              `json:"scForNFSPvc,omitempty"`
	SCForNFSProvisioner   string              `json:"scForNFS,omitempty"`
	MountOptions          []string            `json:"mountOptions,omitempty"`
}

// NFSProvisionerStatus defines the observed state of NFSProvisioner
type NFSProvisionerStatus struct {
	Phase              string             `json:"phase,omitempty"`
	Error              string             `json:"error,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	Nodes              []string           `json:"nodes,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
}

// ImageConfiguration holds configuration of the image to use
type ImageConfiguration struct {
	// Set nfs provisioner operator image
	// +kubebuilder:default="k8s.gcr.io/sig-storage/nfs-provisioner@sha256:e943bb77c7df05ebdc8c7888b2db289b13bf9f012d6a3a5a74f14d4d5743d439"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="NFS Provisioner Image",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Image *string `json:"image"`
	// Image PullPolicy is for nfs provisioner operator image.
	// +kubebuilder:default="IfNotPresent"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Pull Policy",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:imagePullPolicy"}
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NFSProvisioner is the Schema for the nfsprovisioners API
// +operator-sdk:csv:customresourcedefinitions:displayName="NFS Provisioner App",resources={{ServiceAccount,v1,nfs-provisioner},{SecurityContextConstraints,v1,nfs-provisioner},{Deployment,v1,nfs-provisioner},{PersistentVolumeClaim,v1,nfs-server},{ClusterRole,v1,nfs-provisioner-runner},{ClusterRoleBinding,v1,nfs-provisioner-runner},{Role,v1,leader-locking-nfs-provisioner},{RoleBinding,v1,leader-locking-nfs-provisioner},{Service,v1,nfs-provisioner},{StorageClass,v1,nfs}}
type NFSProvisioner struct {
	Spec              NFSProvisionerSpec `json:"spec,omitempty"`
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            NFSProvisionerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NFSProvisionerList contains a list of NFSProvisioner
type NFSProvisionerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NFSProvisioner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NFSProvisioner{}, &NFSProvisionerList{})
}

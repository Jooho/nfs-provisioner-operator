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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NFSProvisionerSpec defines the desired state of NFSProvisioner
type NFSProvisionerSpec struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Pvc string `json:"pvc,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Action-Items
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	SCForNFSPvc string `json:"scForNFSPvc,omitempty"` //https://golang.org/pkg/encoding/json/
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	SCForNFSProvisioner string `json:"scForNFS,omitempty"` //https://golang.org/pkg/encoding/json/

}

// NFSProvisionerStatus defines the observed state of NFSProvisioner
type NFSProvisionerStatus struct {

	// Nodes are the names of the NFS pods
	Nodes []string `json:"nodes"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NFSProvisioner is the Schema for the nfsprovisioners API
// +operator-sdk:csv:customresourcedefinitions:displayName="NFS Provisioner App",resources={{ServiceAccount,v1,nfs-provisioner},{SecurityContextConstraints,v1,nfs-provisioner},{Deployment,v1,nfs-provisioner},{PersistentVolumeClaim,v1,nfs-server},{ClusterRole,v1,nfs-provisioner-runner},{ClusterRoleBinding,v1,nfs-provisioner-runner},{Role,v1,leader-locking-nfs-provisioner},{RoleBinding,v1,leader-locking-nfs-provisioner},{Service,v1,nfs-provisioner},{StorageClass,v1,nfs}}
type NFSProvisioner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NFSProvisionerSpec   `json:"spec,omitempty"`
	Status NFSProvisionerStatus `json:"status,omitempty"`
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

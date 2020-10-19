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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NFSProvisionerSpec defines the desired state of NFSProvisioner
type NFSProvisionerSpec struct {
	Pvc     string            `json:"pvc,omitempty"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Action-Items
	SCForNFSPvc string `json:"scForNFSPvc,omitempty"` //https://golang.org/pkg/encoding/json/
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


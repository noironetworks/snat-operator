// Copyright 2019 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the License);
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an AS IS BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SnatGlobalInfoSpec defines the desired state of SnatGlobalInfo
// +k8s:openapi-gen=true
type SnatGlobalInfoSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	GlobalInfos map[string][]GlobalInfo `json:"globalInfos,omitempty"`
}

// SnatGlobalInfoStatus defines the observed state of SnatGlobalInfo
// +k8s:openapi-gen=true
type SnatGlobalInfoStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SnatGlobalInfo is the Schema for the snatglobalinfos API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type SnatGlobalInfo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SnatGlobalInfoSpec   `json:"spec,omitempty"`
	Status SnatGlobalInfoStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SnatGlobalInfoList contains a list of SnatGlobalInfo
type SnatGlobalInfoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SnatGlobalInfo `json:"items"`
}

type GlobalInfo struct {
	MacAddress string      `json:"macAddress"`
	PortRanges []PortRange `json:"portRanges"`
	SnatIp     string      `json:"snatIp"`
	SnatIpUid  string      `json:"snatIpUid"`
	// +kubebuilder:validation:Enum=tcp,udp,icmp
	Protocols []string `json:"protocols"`
}

// +k8s:openapi-gen=true
type PortRange struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Start int `json:"start,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	End int `json:"end,omitempty"`
}

func init() {
	SchemeBuilder.Register(&SnatGlobalInfo{}, &SnatGlobalInfoList{})
}

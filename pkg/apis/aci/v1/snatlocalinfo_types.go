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

// SnatLocalInfoSpec defines the desired state of SnatLocalInfo
// +k8s:openapi-gen=true
type SnatLocalInfoSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	LocalInfos map[string]LocalInfo `json:"localInfos,omitempty"`
}

// SnatLocalInfoStatus defines the observed state of SnatLocalInfo
// +k8s:openapi-gen=true
type SnatLocalInfoStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Status string `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SnatLocalInfo is the Schema for the snatlocalinfos API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type SnatLocalInfo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SnatLocalInfoSpec   `json:"spec,omitempty"`
	Status SnatLocalInfoStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SnatLocalInfoList contains a list of SnatLocalInfo
type SnatLocalInfoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SnatLocalInfo `json:"items"`
}

type LocalInfo struct {
	PodName        string `json:"podName"`
	PodNamespace   string `json:"podNamespace"`
	SnatIp         string `json:"snatIp"`
	SnatPolicyName string `json:"snatPolicyName"`
	SnatScope      string `json:"resourcetype"`
}

func init() {
	SchemeBuilder.Register(&SnatLocalInfo{}, &SnatLocalInfoList{})
}

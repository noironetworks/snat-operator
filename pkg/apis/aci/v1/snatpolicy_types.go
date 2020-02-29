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

type PolicyState string

const (
	Ready            PolicyState = "Ready"
	Failed           PolicyState = "Failed"
	IpPortsExhausted PolicyState = "IpPortsExhausted"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SnatPolicySpec defines the desired state of SnatPolicy
// +k8s:openapi-gen=true
type SnatPolicySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	SnatIp   []string    `json:"snatIp"`
	Selector PodSelector `json:"selector,omitempty"`
}

// SnatPolicyStatus defines the observed state of SnatPolicy
// +k8s:openapi-gen=true
// +genclient:nonNamespaced
type SnatPolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file7
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	SnatPortsAllocated map[string][]NodePortRange `json:"snatPortsAllocated,omitempty"`
	State              PolicyState                `json:"state"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SnatPolicy is the Schema for the snatpolicies API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type SnatPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SnatPolicySpec   `json:"spec,omitempty"`
	Status SnatPolicyStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SnatPolicyList contains a list of SnatPolicy
type SnatPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SnatPolicy `json:"items"`
}

type PodSelector struct {
	Labels    map[string]string `json:"labels,omitempty"`
	Namespace string            `json:"namespace,omitempty"`
}
type NodePortRange struct {
	NodeName  string    `json:"nodename,omitempty"`
	PortRange PortRange `json:"portrange,omitempty"`
}

func init() {
	SchemeBuilder.Register(&SnatPolicy{}, &SnatPolicyList{})
}

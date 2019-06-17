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

func init() {
	SchemeBuilder.Register(&SnatGlobalInfo{}, &SnatGlobalInfoList{})
}
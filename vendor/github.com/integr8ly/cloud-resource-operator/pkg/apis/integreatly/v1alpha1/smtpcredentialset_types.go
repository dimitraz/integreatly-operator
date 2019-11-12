package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SMTPCredentialsSpec defines the desired state of SMTPCredentials
// +k8s:openapi-gen=true
type SMTPCredentialSetSpec ResourceTypeSpec

// SMTPCredentialsStatus defines the observed state of SMTPCredentials
// +k8s:openapi-gen=true
type SMTPCredentialSetStatus ResourceTypeStatus

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SMTPCredentials is the Schema for the smtpcredentialset API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type SMTPCredentialSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SMTPCredentialSetSpec   `json:"spec,omitempty"`
	Status SMTPCredentialSetStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SMTPCredentialsList contains a list of SMTPCredentials
type SMTPCredentialSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SMTPCredentialSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SMTPCredentialSet{}, &SMTPCredentialSetList{})
}

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NginxSpec defines the desired state of a Nginx.
type NginxSpec struct {
	// Nginx Version to deploy
	Version string `json:"version"`
}

// NginxStatus defines the observed state of a Nginx
type NginxStatus struct {
	// The most recent generation observed by the controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions is a list of status conditions ths object is in.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// DEPRECATED: This field is not part of any API contract
	// it will go away as soon as kubectl can print conditions!
	// Human readable status - please use .Conditions from code
	Phase NginxPhase `json:"phase,omitempty"`
}

const (
	NginxAvailable = "Available"
)

type NginxPhase string

// Well-known Nginx Phases for printing a Status in kubectl,
// see deprecation notice in NginxStatus for details.
const (
	NginxPhasePending  NginxPhase = "Pending"
	NginxPhaseReady    NginxPhase = "Ready"
	NginxPhaseNotReady NginxPhase = "NotReady"
)

// Nginx controls the handover process between two operators.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Nginx struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NginxSpec `json:"spec,omitempty"`
	// +kubebuilder:default={phase:Pending}
	Status NginxStatus `json:"status,omitempty"`
}

// NginxList contains a list of Nginxs
// +kubebuilder:object:root=true
type NginxList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Nginx `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Nginx{}, &NginxList{})
}

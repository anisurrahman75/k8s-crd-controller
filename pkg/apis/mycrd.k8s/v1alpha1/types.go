// +groupName=mycrd.k8s
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppsCode is a specification for a AppsCode resource
type AppsCode struct {
	metav1.TypeMeta   `json:"metav1.TypeMeta"`
	metav1.ObjectMeta `json:"metav1.ObjectMeta"`

	Spec   AppsCodeSpec   `json:"spec"`
	Status AppsCodeStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppCodeList is the List of the AppsCode resources
type AppsCodeList struct {
	metav1.TypeMeta `json:"metav1.TypeMeta"`
	metav1.ListMeta `json:"metav1.ListMeta"`

	Items []AppsCode `json:"items"`
}

// AppsCodeSpec is the spec for a AppsCode Resource
type AppsCodeSpec struct {
	Name     string `json:"name"`
	Replicas *int32 `json:"replicas"`
	Image    string `json:"image"`
	Port     int32  `json:"port"`
}

// AppsCode Status Is the Status of the AppsCode Resources
type AppsCodeStatus struct {
	AvailableReplicas int32 `json:"availableReplicas"`
}

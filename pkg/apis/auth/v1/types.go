package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AuthBackend represents a auth backend.
//
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="NAME",type="string",JSONPath=".metadata.name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:singular=authbackend,path=authbackends,shortName=ab,scope=Cluster,categories=auth;bifrost
type AuthBackend struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuthBackendSpec   `json:"spec,omitempty"`
	Status AuthBackendStatus `json:"status,omitempty"`
}

// AuthBackendSpec is the spec of an auth backend.
type AuthBackendSpec struct {
	Dex *AuthBackendDex `json:"dex,omitempty"`
}

// AuthBackendDex is the spec for a Dex based auth backend.
type AuthBackendDex struct {
}

// AuthBackendStatus is the auth backend  status
type AuthBackendStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AuthBackendList is a list of AuthBackends resources.
type AuthBackendList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AuthBackend `json:"items"`
}

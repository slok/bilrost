package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AuthBackend represents a auth backend.
//
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:singular=authbackend,path=authbackends,shortName=ab,scope=Cluster,categories=auth;bilrost
type AuthBackend struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuthBackendSpec   `json:"spec,omitempty"`
	Status AuthBackendStatus `json:"status,omitempty"`
}

// AuthBackendSpec is the spec of an auth backend.
type AuthBackendSpec struct {
	AuthBackendSource `json:",inline"`
}

// AuthBackendSource has the configuration of the auth backends.
type AuthBackendSource struct {
	Dex *AuthBackendDex `json:"dex,omitempty"`
}

// AuthBackendDex is the spec for a Dex based auth backend.
type AuthBackendDex struct {
	PublicURL  string `json:"publicURL"`
	APIAddress string `json:"apiAddress"`
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

// IngressAuth represents a auth configuraiton for an ingress.
//
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:singular=ingressauth,path=ingressauths,shortName=ia,scope=Namespaced,categories=auth;bilrost
type IngressAuth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IngressAuthSpec   `json:"spec,omitempty"`
	Status IngressAuthStatus `json:"status,omitempty"`
}

// IngressAuthSpec is the spec of an auth backend.
type IngressAuthSpec struct {
	AuthSettings    AuthSettings `json:"authSettings,omitempty"`
	AuthProxySource `json:",inline"`
}

// AuthProxySource has the auth proxies configuration.
type AuthProxySource struct {
	Oauth2Proxy *Oauth2ProxyAuthProxySource `json:"oauth2Proxy,omitempty"`
}

// AuthSettings are the Oauth2 and/or OIDC settings.
type AuthSettings struct {
	ScopeOrClaims []string `json:"scopeOrClaims,omitempty"`
}

// Oauth2ProxyAuthProxySource has the configuration of an oauth2proxy
type Oauth2ProxyAuthProxySource struct {
	CommonProxySettings `json:",inline"`
}

// CommonProxySettings are settings that all proxies will have.
type CommonProxySettings struct {
	Image     string                       `json:"image,omitempty"`
	Replicas  int                          `json:"replicas,omitempty"`
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// IngressAuthStatus is the ingress auth status.
type IngressAuthStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IngressAuthList is a list of IngressAuths resources.
type IngressAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []IngressAuth `json:"items"`
}

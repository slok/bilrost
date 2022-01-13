//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1

import (
	corev1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthBackend) DeepCopyInto(out *AuthBackend) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthBackend.
func (in *AuthBackend) DeepCopy() *AuthBackend {
	if in == nil {
		return nil
	}
	out := new(AuthBackend)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AuthBackend) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthBackendDex) DeepCopyInto(out *AuthBackendDex) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthBackendDex.
func (in *AuthBackendDex) DeepCopy() *AuthBackendDex {
	if in == nil {
		return nil
	}
	out := new(AuthBackendDex)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthBackendList) DeepCopyInto(out *AuthBackendList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AuthBackend, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthBackendList.
func (in *AuthBackendList) DeepCopy() *AuthBackendList {
	if in == nil {
		return nil
	}
	out := new(AuthBackendList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AuthBackendList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthBackendSource) DeepCopyInto(out *AuthBackendSource) {
	*out = *in
	if in.Dex != nil {
		in, out := &in.Dex, &out.Dex
		*out = new(AuthBackendDex)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthBackendSource.
func (in *AuthBackendSource) DeepCopy() *AuthBackendSource {
	if in == nil {
		return nil
	}
	out := new(AuthBackendSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthBackendSpec) DeepCopyInto(out *AuthBackendSpec) {
	*out = *in
	in.AuthBackendSource.DeepCopyInto(&out.AuthBackendSource)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthBackendSpec.
func (in *AuthBackendSpec) DeepCopy() *AuthBackendSpec {
	if in == nil {
		return nil
	}
	out := new(AuthBackendSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthBackendStatus) DeepCopyInto(out *AuthBackendStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthBackendStatus.
func (in *AuthBackendStatus) DeepCopy() *AuthBackendStatus {
	if in == nil {
		return nil
	}
	out := new(AuthBackendStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthProxySource) DeepCopyInto(out *AuthProxySource) {
	*out = *in
	if in.Oauth2Proxy != nil {
		in, out := &in.Oauth2Proxy, &out.Oauth2Proxy
		*out = new(Oauth2ProxyAuthProxySource)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthProxySource.
func (in *AuthProxySource) DeepCopy() *AuthProxySource {
	if in == nil {
		return nil
	}
	out := new(AuthProxySource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthSettings) DeepCopyInto(out *AuthSettings) {
	*out = *in
	if in.ScopeOrClaims != nil {
		in, out := &in.ScopeOrClaims, &out.ScopeOrClaims
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthSettings.
func (in *AuthSettings) DeepCopy() *AuthSettings {
	if in == nil {
		return nil
	}
	out := new(AuthSettings)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CommonProxySettings) DeepCopyInto(out *CommonProxySettings) {
	*out = *in
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = new(corev1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CommonProxySettings.
func (in *CommonProxySettings) DeepCopy() *CommonProxySettings {
	if in == nil {
		return nil
	}
	out := new(CommonProxySettings)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressAuth) DeepCopyInto(out *IngressAuth) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressAuth.
func (in *IngressAuth) DeepCopy() *IngressAuth {
	if in == nil {
		return nil
	}
	out := new(IngressAuth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IngressAuth) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressAuthList) DeepCopyInto(out *IngressAuthList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]IngressAuth, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressAuthList.
func (in *IngressAuthList) DeepCopy() *IngressAuthList {
	if in == nil {
		return nil
	}
	out := new(IngressAuthList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IngressAuthList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressAuthSpec) DeepCopyInto(out *IngressAuthSpec) {
	*out = *in
	in.AuthSettings.DeepCopyInto(&out.AuthSettings)
	in.AuthProxySource.DeepCopyInto(&out.AuthProxySource)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressAuthSpec.
func (in *IngressAuthSpec) DeepCopy() *IngressAuthSpec {
	if in == nil {
		return nil
	}
	out := new(IngressAuthSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressAuthStatus) DeepCopyInto(out *IngressAuthStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressAuthStatus.
func (in *IngressAuthStatus) DeepCopy() *IngressAuthStatus {
	if in == nil {
		return nil
	}
	out := new(IngressAuthStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Oauth2ProxyAuthProxySource) DeepCopyInto(out *Oauth2ProxyAuthProxySource) {
	*out = *in
	in.CommonProxySettings.DeepCopyInto(&out.CommonProxySettings)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Oauth2ProxyAuthProxySource.
func (in *Oauth2ProxyAuthProxySource) DeepCopy() *Oauth2ProxyAuthProxySource {
	if in == nil {
		return nil
	}
	out := new(Oauth2ProxyAuthProxySource)
	in.DeepCopyInto(out)
	return out
}

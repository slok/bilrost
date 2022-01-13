// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"context"
	"time"

	v1 "github.com/slok/bilrost/pkg/apis/auth/v1"
	scheme "github.com/slok/bilrost/pkg/kubernetes/gen/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// AuthBackendsGetter has a method to return a AuthBackendInterface.
// A group's client should implement this interface.
type AuthBackendsGetter interface {
	AuthBackends() AuthBackendInterface
}

// AuthBackendInterface has methods to work with AuthBackend resources.
type AuthBackendInterface interface {
	Create(ctx context.Context, authBackend *v1.AuthBackend, opts metav1.CreateOptions) (*v1.AuthBackend, error)
	Update(ctx context.Context, authBackend *v1.AuthBackend, opts metav1.UpdateOptions) (*v1.AuthBackend, error)
	UpdateStatus(ctx context.Context, authBackend *v1.AuthBackend, opts metav1.UpdateOptions) (*v1.AuthBackend, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.AuthBackend, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.AuthBackendList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.AuthBackend, err error)
	AuthBackendExpansion
}

// authBackends implements AuthBackendInterface
type authBackends struct {
	client rest.Interface
}

// newAuthBackends returns a AuthBackends
func newAuthBackends(c *AuthV1Client) *authBackends {
	return &authBackends{
		client: c.RESTClient(),
	}
}

// Get takes name of the authBackend, and returns the corresponding authBackend object, and an error if there is any.
func (c *authBackends) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.AuthBackend, err error) {
	result = &v1.AuthBackend{}
	err = c.client.Get().
		Resource("authbackends").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of AuthBackends that match those selectors.
func (c *authBackends) List(ctx context.Context, opts metav1.ListOptions) (result *v1.AuthBackendList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.AuthBackendList{}
	err = c.client.Get().
		Resource("authbackends").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested authBackends.
func (c *authBackends) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("authbackends").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a authBackend and creates it.  Returns the server's representation of the authBackend, and an error, if there is any.
func (c *authBackends) Create(ctx context.Context, authBackend *v1.AuthBackend, opts metav1.CreateOptions) (result *v1.AuthBackend, err error) {
	result = &v1.AuthBackend{}
	err = c.client.Post().
		Resource("authbackends").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(authBackend).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a authBackend and updates it. Returns the server's representation of the authBackend, and an error, if there is any.
func (c *authBackends) Update(ctx context.Context, authBackend *v1.AuthBackend, opts metav1.UpdateOptions) (result *v1.AuthBackend, err error) {
	result = &v1.AuthBackend{}
	err = c.client.Put().
		Resource("authbackends").
		Name(authBackend.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(authBackend).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *authBackends) UpdateStatus(ctx context.Context, authBackend *v1.AuthBackend, opts metav1.UpdateOptions) (result *v1.AuthBackend, err error) {
	result = &v1.AuthBackend{}
	err = c.client.Put().
		Resource("authbackends").
		Name(authBackend.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(authBackend).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the authBackend and deletes it. Returns an error if one occurs.
func (c *authBackends) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("authbackends").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *authBackends) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("authbackends").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched authBackend.
func (c *authBackends) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.AuthBackend, err error) {
	result = &v1.AuthBackend{}
	err = c.client.Patch(pt).
		Resource("authbackends").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

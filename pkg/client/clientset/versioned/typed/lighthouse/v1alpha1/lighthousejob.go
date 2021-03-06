// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	scheme "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// LighthouseJobsGetter has a method to return a LighthouseJobInterface.
// A group's client should implement this interface.
type LighthouseJobsGetter interface {
	LighthouseJobs(namespace string) LighthouseJobInterface
}

// LighthouseJobInterface has methods to work with LighthouseJob resources.
type LighthouseJobInterface interface {
	Create(ctx context.Context, lighthouseJob *v1alpha1.LighthouseJob, opts v1.CreateOptions) (*v1alpha1.LighthouseJob, error)
	Update(ctx context.Context, lighthouseJob *v1alpha1.LighthouseJob, opts v1.UpdateOptions) (*v1alpha1.LighthouseJob, error)
	UpdateStatus(ctx context.Context, lighthouseJob *v1alpha1.LighthouseJob, opts v1.UpdateOptions) (*v1alpha1.LighthouseJob, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.LighthouseJob, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.LighthouseJobList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.LighthouseJob, err error)
	LighthouseJobExpansion
}

// lighthouseJobs implements LighthouseJobInterface
type lighthouseJobs struct {
	client rest.Interface
	ns     string
}

// newLighthouseJobs returns a LighthouseJobs
func newLighthouseJobs(c *LighthouseV1alpha1Client, namespace string) *lighthouseJobs {
	return &lighthouseJobs{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the lighthouseJob, and returns the corresponding lighthouseJob object, and an error if there is any.
func (c *lighthouseJobs) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.LighthouseJob, err error) {
	result = &v1alpha1.LighthouseJob{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("lighthousejobs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of LighthouseJobs that match those selectors.
func (c *lighthouseJobs) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.LighthouseJobList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.LighthouseJobList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("lighthousejobs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested lighthouseJobs.
func (c *lighthouseJobs) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("lighthousejobs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a lighthouseJob and creates it.  Returns the server's representation of the lighthouseJob, and an error, if there is any.
func (c *lighthouseJobs) Create(ctx context.Context, lighthouseJob *v1alpha1.LighthouseJob, opts v1.CreateOptions) (result *v1alpha1.LighthouseJob, err error) {
	result = &v1alpha1.LighthouseJob{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("lighthousejobs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(lighthouseJob).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a lighthouseJob and updates it. Returns the server's representation of the lighthouseJob, and an error, if there is any.
func (c *lighthouseJobs) Update(ctx context.Context, lighthouseJob *v1alpha1.LighthouseJob, opts v1.UpdateOptions) (result *v1alpha1.LighthouseJob, err error) {
	result = &v1alpha1.LighthouseJob{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("lighthousejobs").
		Name(lighthouseJob.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(lighthouseJob).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *lighthouseJobs) UpdateStatus(ctx context.Context, lighthouseJob *v1alpha1.LighthouseJob, opts v1.UpdateOptions) (result *v1alpha1.LighthouseJob, err error) {
	result = &v1alpha1.LighthouseJob{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("lighthousejobs").
		Name(lighthouseJob.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(lighthouseJob).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the lighthouseJob and deletes it. Returns an error if one occurs.
func (c *lighthouseJobs) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("lighthousejobs").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *lighthouseJobs) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("lighthousejobs").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched lighthouseJob.
func (c *lighthouseJobs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.LighthouseJob, err error) {
	result = &v1alpha1.LighthouseJob{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("lighthousejobs").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

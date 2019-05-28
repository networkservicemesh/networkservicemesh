// Copyright (c) 2019 Cisco and/or its affiliates.
// Copyright (c) 2019 Red Hat Inc. and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"time"

	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	scheme "github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// NetworkServiceManagersGetter has a method to return a NetworkServiceManagerInterface.
// A group's client should implement this interface.
type NetworkServiceManagersGetter interface {
	NetworkServiceManagers(namespace string) NetworkServiceManagerInterface
}

// NetworkServiceManagerInterface has methods to work with NetworkServiceManager resources.
type NetworkServiceManagerInterface interface {
	Create(*v1.NetworkServiceManager) (*v1.NetworkServiceManager, error)
	Update(*v1.NetworkServiceManager) (*v1.NetworkServiceManager, error)
	UpdateStatus(*v1.NetworkServiceManager) (*v1.NetworkServiceManager, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error
	Get(name string, options metav1.GetOptions) (*v1.NetworkServiceManager, error)
	List(opts metav1.ListOptions) (*v1.NetworkServiceManagerList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.NetworkServiceManager, err error)
	NetworkServiceManagerExpansion
}

// networkServiceManagers implements NetworkServiceManagerInterface
type networkServiceManagers struct {
	client rest.Interface
	ns     string
}

// newNetworkServiceManagers returns a NetworkServiceManagers
func newNetworkServiceManagers(c *NetworkservicemeshV1Client, namespace string) *networkServiceManagers {
	return &networkServiceManagers{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the networkServiceManager, and returns the corresponding networkServiceManager object, and an error if there is any.
func (c *networkServiceManagers) Get(name string, options metav1.GetOptions) (result *v1.NetworkServiceManager, err error) {
	result = &v1.NetworkServiceManager{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("networkservicemanagers").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of NetworkServiceManagers that match those selectors.
func (c *networkServiceManagers) List(opts metav1.ListOptions) (result *v1.NetworkServiceManagerList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.NetworkServiceManagerList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("networkservicemanagers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested networkServiceManagers.
func (c *networkServiceManagers) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("networkservicemanagers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a networkServiceManager and creates it.  Returns the server's representation of the networkServiceManager, and an error, if there is any.
func (c *networkServiceManagers) Create(networkServiceManager *v1.NetworkServiceManager) (result *v1.NetworkServiceManager, err error) {
	result = &v1.NetworkServiceManager{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("networkservicemanagers").
		Body(networkServiceManager).
		Do().
		Into(result)
	return
}

// Update takes the representation of a networkServiceManager and updates it. Returns the server's representation of the networkServiceManager, and an error, if there is any.
func (c *networkServiceManagers) Update(networkServiceManager *v1.NetworkServiceManager) (result *v1.NetworkServiceManager, err error) {
	result = &v1.NetworkServiceManager{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("networkservicemanagers").
		Name(networkServiceManager.Name).
		Body(networkServiceManager).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *networkServiceManagers) UpdateStatus(networkServiceManager *v1.NetworkServiceManager) (result *v1.NetworkServiceManager, err error) {
	result = &v1.NetworkServiceManager{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("networkservicemanagers").
		Name(networkServiceManager.Name).
		SubResource("status").
		Body(networkServiceManager).
		Do().
		Into(result)
	return
}

// Delete takes name of the networkServiceManager and deletes it. Returns an error if one occurs.
func (c *networkServiceManagers) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("networkservicemanagers").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *networkServiceManagers) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("networkservicemanagers").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched networkServiceManager.
func (c *networkServiceManagers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.NetworkServiceManager, err error) {
	result = &v1.NetworkServiceManager{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("networkservicemanagers").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}

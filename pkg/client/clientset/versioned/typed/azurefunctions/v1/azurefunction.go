/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/yaron2/azfuncs/pkg/apis/azurefunctions/v1"
	scheme "github.com/yaron2/azfuncs/pkg/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// AzureFunctionsGetter has a method to return a AzureFunctionInterface.
// A group's client should implement this interface.
type AzureFunctionsGetter interface {
	AzureFunctions(namespace string) AzureFunctionInterface
}

// AzureFunctionInterface has methods to work with AzureFunction resources.
type AzureFunctionInterface interface {
	Create(*v1.AzureFunction) (*v1.AzureFunction, error)
	Update(*v1.AzureFunction) (*v1.AzureFunction, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error
	Get(name string, options metav1.GetOptions) (*v1.AzureFunction, error)
	List(opts metav1.ListOptions) (*v1.AzureFunctionList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.AzureFunction, err error)
	AzureFunctionExpansion
}

// azureFunctions implements AzureFunctionInterface
type azureFunctions struct {
	client rest.Interface
	ns     string
}

// newAzureFunctions returns a AzureFunctions
func newAzureFunctions(c *DevV1Client, namespace string) *azureFunctions {
	return &azureFunctions{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the azureFunction, and returns the corresponding azureFunction object, and an error if there is any.
func (c *azureFunctions) Get(name string, options metav1.GetOptions) (result *v1.AzureFunction, err error) {
	result = &v1.AzureFunction{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("azurefunctions").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of AzureFunctions that match those selectors.
func (c *azureFunctions) List(opts metav1.ListOptions) (result *v1.AzureFunctionList, err error) {
	result = &v1.AzureFunctionList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("azurefunctions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested azureFunctions.
func (c *azureFunctions) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("azurefunctions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a azureFunction and creates it.  Returns the server's representation of the azureFunction, and an error, if there is any.
func (c *azureFunctions) Create(azureFunction *v1.AzureFunction) (result *v1.AzureFunction, err error) {
	result = &v1.AzureFunction{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("azurefunctions").
		Body(azureFunction).
		Do().
		Into(result)
	return
}

// Update takes the representation of a azureFunction and updates it. Returns the server's representation of the azureFunction, and an error, if there is any.
func (c *azureFunctions) Update(azureFunction *v1.AzureFunction) (result *v1.AzureFunction, err error) {
	result = &v1.AzureFunction{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("azurefunctions").
		Name(azureFunction.Name).
		Body(azureFunction).
		Do().
		Into(result)
	return
}

// Delete takes name of the azureFunction and deletes it. Returns an error if one occurs.
func (c *azureFunctions) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("azurefunctions").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *azureFunctions) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("azurefunctions").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched azureFunction.
func (c *azureFunctions) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.AzureFunction, err error) {
	result = &v1.AzureFunction{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("azurefunctions").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}

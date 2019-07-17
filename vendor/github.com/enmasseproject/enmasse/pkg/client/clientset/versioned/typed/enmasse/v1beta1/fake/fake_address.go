/*
 * Copyright 2018-2019, EnMasse authors.
 * License: Apache License 2.0 (see the file LICENSE or http://apache.org/licenses/LICENSE-2.0.html).
 */

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1beta1 "github.com/enmasseproject/enmasse/pkg/apis/enmasse/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeAddresses implements AddressInterface
type FakeAddresses struct {
	Fake *FakeEnmasseV1beta1
	ns   string
}

var addressesResource = schema.GroupVersionResource{Group: "enmasse.io", Version: "v1beta1", Resource: "addresses"}

var addressesKind = schema.GroupVersionKind{Group: "enmasse.io", Version: "v1beta1", Kind: "Address"}

// Get takes name of the address, and returns the corresponding address object, and an error if there is any.
func (c *FakeAddresses) Get(name string, options v1.GetOptions) (result *v1beta1.Address, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(addressesResource, c.ns, name), &v1beta1.Address{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.Address), err
}

// List takes label and field selectors, and returns the list of Addresses that match those selectors.
func (c *FakeAddresses) List(opts v1.ListOptions) (result *v1beta1.AddressList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(addressesResource, addressesKind, c.ns, opts), &v1beta1.AddressList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.AddressList{ListMeta: obj.(*v1beta1.AddressList).ListMeta}
	for _, item := range obj.(*v1beta1.AddressList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested addresses.
func (c *FakeAddresses) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(addressesResource, c.ns, opts))

}

// Create takes the representation of a address and creates it.  Returns the server's representation of the address, and an error, if there is any.
func (c *FakeAddresses) Create(address *v1beta1.Address) (result *v1beta1.Address, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(addressesResource, c.ns, address), &v1beta1.Address{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.Address), err
}

// Update takes the representation of a address and updates it. Returns the server's representation of the address, and an error, if there is any.
func (c *FakeAddresses) Update(address *v1beta1.Address) (result *v1beta1.Address, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(addressesResource, c.ns, address), &v1beta1.Address{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.Address), err
}

// Delete takes name of the address and deletes it. Returns an error if one occurs.
func (c *FakeAddresses) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(addressesResource, c.ns, name), &v1beta1.Address{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAddresses) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(addressesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1beta1.AddressList{})
	return err
}

// Patch applies the patch and returns the patched address.
func (c *FakeAddresses) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.Address, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(addressesResource, c.ns, name, data, subresources...), &v1beta1.Address{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.Address), err
}
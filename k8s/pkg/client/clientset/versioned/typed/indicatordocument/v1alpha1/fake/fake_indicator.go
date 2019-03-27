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

package fake

import (
	v1alpha1 "github.com/pivotal/monitoring-indicator-protocol/k8s/pkg/apis/indicatordocument/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeIndicators implements IndicatorInterface
type FakeIndicators struct {
	Fake *FakeAppsV1alpha1
	ns   string
}

var indicatorsResource = schema.GroupVersionResource{Group: "apps.pivotal.io", Version: "v1alpha1", Resource: "indicators"}

var indicatorsKind = schema.GroupVersionKind{Group: "apps.pivotal.io", Version: "v1alpha1", Kind: "Indicator"}

// Get takes name of the indicator, and returns the corresponding indicator object, and an error if there is any.
func (c *FakeIndicators) Get(name string, options v1.GetOptions) (result *v1alpha1.Indicator, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(indicatorsResource, c.ns, name), &v1alpha1.Indicator{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Indicator), err
}

// List takes label and field selectors, and returns the list of Indicators that match those selectors.
func (c *FakeIndicators) List(opts v1.ListOptions) (result *v1alpha1.IndicatorList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(indicatorsResource, indicatorsKind, c.ns, opts), &v1alpha1.IndicatorList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.IndicatorList{ListMeta: obj.(*v1alpha1.IndicatorList).ListMeta}
	for _, item := range obj.(*v1alpha1.IndicatorList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested indicators.
func (c *FakeIndicators) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(indicatorsResource, c.ns, opts))

}

// Create takes the representation of a indicator and creates it.  Returns the server's representation of the indicator, and an error, if there is any.
func (c *FakeIndicators) Create(indicator *v1alpha1.Indicator) (result *v1alpha1.Indicator, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(indicatorsResource, c.ns, indicator), &v1alpha1.Indicator{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Indicator), err
}

// Update takes the representation of a indicator and updates it. Returns the server's representation of the indicator, and an error, if there is any.
func (c *FakeIndicators) Update(indicator *v1alpha1.Indicator) (result *v1alpha1.Indicator, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(indicatorsResource, c.ns, indicator), &v1alpha1.Indicator{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Indicator), err
}

// Delete takes name of the indicator and deletes it. Returns an error if one occurs.
func (c *FakeIndicators) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(indicatorsResource, c.ns, name), &v1alpha1.Indicator{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeIndicators) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(indicatorsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.IndicatorList{})
	return err
}

// Patch applies the patch and returns the patched indicator.
func (c *FakeIndicators) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Indicator, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(indicatorsResource, c.ns, name, pt, data, subresources...), &v1alpha1.Indicator{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Indicator), err
}

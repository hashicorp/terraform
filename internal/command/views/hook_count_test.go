// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"reflect"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"

	legacy "github.com/hashicorp/terraform/internal/legacy/terraform"
)

func testCountHookResourceID(addr addrs.AbsResourceInstance) terraform.HookResourceIdentity {
	return terraform.HookResourceIdentity{
		Addr: addr,
		ProviderAddr: addrs.Provider{
			Type:      "test",
			Namespace: "hashicorp",
			Hostname:  "example.com",
		},
	}
}

func TestCountHook_impl(t *testing.T) {
	var _ terraform.Hook = new(countHook)
}

func TestCountHookPostDiff_DestroyDeposed(t *testing.T) {
	h := new(countHook)

	resources := map[string]*legacy.InstanceDiff{
		"lorem": {DestroyDeposed: true},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PostDiff(testCountHookResourceID(addr), states.DeposedKey("deadbeef"), plans.Delete, cty.DynamicVal, cty.DynamicVal)
	}

	expected := new(countHook)
	expected.ToAdd = 0
	expected.ToChange = 0
	expected.ToRemoveAndAdd = 0
	expected.ToRemove = 1

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected %#v, got %#v instead.", expected, h)
	}
}

func TestCountHookPostDiff_DestroyOnly(t *testing.T) {
	h := new(countHook)

	resources := map[string]*legacy.InstanceDiff{
		"foo":   {Destroy: true},
		"bar":   {Destroy: true},
		"lorem": {Destroy: true},
		"ipsum": {Destroy: true},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PostDiff(testCountHookResourceID(addr), addrs.NotDeposed, plans.Delete, cty.DynamicVal, cty.DynamicVal)
	}

	expected := new(countHook)
	expected.ToAdd = 0
	expected.ToChange = 0
	expected.ToRemoveAndAdd = 0
	expected.ToRemove = 4

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected %#v, got %#v instead.", expected, h)
	}
}

func TestCountHookPostDiff_AddOnly(t *testing.T) {
	h := new(countHook)

	resources := map[string]*legacy.InstanceDiff{
		"foo": {
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": {RequiresNew: true},
			},
		},
		"bar": {
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": {RequiresNew: true},
			},
		},
		"lorem": {
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": {RequiresNew: true},
			},
		},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PostDiff(testCountHookResourceID(addr), addrs.NotDeposed, plans.Create, cty.DynamicVal, cty.DynamicVal)
	}

	expected := new(countHook)
	expected.ToAdd = 3
	expected.ToChange = 0
	expected.ToRemoveAndAdd = 0
	expected.ToRemove = 0

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected %#v, got %#v instead.", expected, h)
	}
}

func TestCountHookPostDiff_ChangeOnly(t *testing.T) {
	h := new(countHook)

	resources := map[string]*legacy.InstanceDiff{
		"foo": {
			Destroy: false,
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": {},
			},
		},
		"bar": {
			Destroy: false,
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": {},
			},
		},
		"lorem": {
			Destroy: false,
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": {},
			},
		},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PostDiff(testCountHookResourceID(addr), addrs.NotDeposed, plans.Update, cty.DynamicVal, cty.DynamicVal)
	}

	expected := new(countHook)
	expected.ToAdd = 0
	expected.ToChange = 3
	expected.ToRemoveAndAdd = 0
	expected.ToRemove = 0

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected %#v, got %#v instead.", expected, h)
	}
}

func TestCountHookPostDiff_Mixed(t *testing.T) {
	h := new(countHook)

	resources := map[string]plans.Action{
		"foo":   plans.Delete,
		"bar":   plans.NoOp,
		"lorem": plans.Update,
		"ipsum": plans.Delete,
	}

	for k, a := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PostDiff(testCountHookResourceID(addr), addrs.NotDeposed, a, cty.DynamicVal, cty.DynamicVal)
	}

	expected := new(countHook)
	expected.ToAdd = 0
	expected.ToChange = 1
	expected.ToRemoveAndAdd = 0
	expected.ToRemove = 2

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected %#v, got %#v instead.",
			expected, h)
	}
}

func TestCountHookPostDiff_NoChange(t *testing.T) {
	h := new(countHook)

	resources := map[string]*legacy.InstanceDiff{
		"foo":   {},
		"bar":   {},
		"lorem": {},
		"ipsum": {},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PostDiff(testCountHookResourceID(addr), addrs.NotDeposed, plans.NoOp, cty.DynamicVal, cty.DynamicVal)
	}

	expected := new(countHook)
	expected.ToAdd = 0
	expected.ToChange = 0
	expected.ToRemoveAndAdd = 0
	expected.ToRemove = 0

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected %#v, got %#v instead.",
			expected, h)
	}
}

func TestCountHookPostDiff_DataSource(t *testing.T) {
	h := new(countHook)

	resources := map[string]plans.Action{
		"foo":   plans.Delete,
		"bar":   plans.NoOp,
		"lorem": plans.Update,
		"ipsum": plans.Delete,
	}

	for k, a := range resources {
		addr := addrs.Resource{
			Mode: addrs.DataResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PostDiff(testCountHookResourceID(addr), addrs.NotDeposed, a, cty.DynamicVal, cty.DynamicVal)
	}

	expected := new(countHook)
	expected.ToAdd = 0
	expected.ToChange = 0
	expected.ToRemoveAndAdd = 0
	expected.ToRemove = 0

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected %#v, got %#v instead.",
			expected, h)
	}
}

func TestCountHookApply_ChangeOnly(t *testing.T) {
	h := new(countHook)

	resources := map[string]*legacy.InstanceDiff{
		"foo": {
			Destroy: false,
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": {},
			},
		},
		"bar": {
			Destroy: false,
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": {},
			},
		},
		"lorem": {
			Destroy: false,
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": {},
			},
		},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PreApply(testCountHookResourceID(addr), addrs.NotDeposed, plans.Update, cty.DynamicVal, cty.DynamicVal)
		h.PostApply(testCountHookResourceID(addr), addrs.NotDeposed, cty.DynamicVal, nil)
	}

	expected := &countHook{pending: make(map[string]plans.Action)}
	expected.Added = 0
	expected.Changed = 3
	expected.Removed = 0

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected:\n%#v\nGot:\n%#v\n", expected, h)
	}
}

func TestCountHookApply_DestroyOnly(t *testing.T) {
	h := new(countHook)

	resources := map[string]*legacy.InstanceDiff{
		"foo":   {Destroy: true},
		"bar":   {Destroy: true},
		"lorem": {Destroy: true},
		"ipsum": {Destroy: true},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PreApply(testCountHookResourceID(addr), addrs.NotDeposed, plans.Delete, cty.DynamicVal, cty.DynamicVal)
		h.PostApply(testCountHookResourceID(addr), addrs.NotDeposed, cty.DynamicVal, nil)
	}

	expected := &countHook{pending: make(map[string]plans.Action)}
	expected.Added = 0
	expected.Changed = 0
	expected.Removed = 4

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected:\n%#v\nGot:\n%#v\n", expected, h)
	}
}

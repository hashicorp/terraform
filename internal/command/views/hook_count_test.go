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

func TestCountHook_impl(t *testing.T) {
	var _ terraform.Hook = new(countHook)
}

func TestCountHookPostDiff_DestroyDeposed(t *testing.T) {
	h := new(countHook)

	resources := map[string]*legacy.InstanceDiff{
		"lorem": &legacy.InstanceDiff{DestroyDeposed: true},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PostDiff(addr, states.DeposedKey("deadbeef"), plans.Delete, cty.DynamicVal, cty.DynamicVal)
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
		"foo":   &legacy.InstanceDiff{Destroy: true},
		"bar":   &legacy.InstanceDiff{Destroy: true},
		"lorem": &legacy.InstanceDiff{Destroy: true},
		"ipsum": &legacy.InstanceDiff{Destroy: true},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PostDiff(addr, states.CurrentGen, plans.Delete, cty.DynamicVal, cty.DynamicVal)
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
		"foo": &legacy.InstanceDiff{
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": &legacy.ResourceAttrDiff{RequiresNew: true},
			},
		},
		"bar": &legacy.InstanceDiff{
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": &legacy.ResourceAttrDiff{RequiresNew: true},
			},
		},
		"lorem": &legacy.InstanceDiff{
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": &legacy.ResourceAttrDiff{RequiresNew: true},
			},
		},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PostDiff(addr, states.CurrentGen, plans.Create, cty.DynamicVal, cty.DynamicVal)
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
		"foo": &legacy.InstanceDiff{
			Destroy: false,
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": &legacy.ResourceAttrDiff{},
			},
		},
		"bar": &legacy.InstanceDiff{
			Destroy: false,
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": &legacy.ResourceAttrDiff{},
			},
		},
		"lorem": &legacy.InstanceDiff{
			Destroy: false,
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": &legacy.ResourceAttrDiff{},
			},
		},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PostDiff(addr, states.CurrentGen, plans.Update, cty.DynamicVal, cty.DynamicVal)
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

		h.PostDiff(addr, states.CurrentGen, a, cty.DynamicVal, cty.DynamicVal)
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
		"foo":   &legacy.InstanceDiff{},
		"bar":   &legacy.InstanceDiff{},
		"lorem": &legacy.InstanceDiff{},
		"ipsum": &legacy.InstanceDiff{},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PostDiff(addr, states.CurrentGen, plans.NoOp, cty.DynamicVal, cty.DynamicVal)
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

		h.PostDiff(addr, states.CurrentGen, a, cty.DynamicVal, cty.DynamicVal)
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
		"foo": &legacy.InstanceDiff{
			Destroy: false,
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": &legacy.ResourceAttrDiff{},
			},
		},
		"bar": &legacy.InstanceDiff{
			Destroy: false,
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": &legacy.ResourceAttrDiff{},
			},
		},
		"lorem": &legacy.InstanceDiff{
			Destroy: false,
			Attributes: map[string]*legacy.ResourceAttrDiff{
				"foo": &legacy.ResourceAttrDiff{},
			},
		},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PreApply(addr, states.CurrentGen, plans.Update, cty.DynamicVal, cty.DynamicVal)
		h.PostApply(addr, states.CurrentGen, cty.DynamicVal, nil)
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
		"foo":   &legacy.InstanceDiff{Destroy: true},
		"bar":   &legacy.InstanceDiff{Destroy: true},
		"lorem": &legacy.InstanceDiff{Destroy: true},
		"ipsum": &legacy.InstanceDiff{Destroy: true},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PreApply(addr, states.CurrentGen, plans.Delete, cty.DynamicVal, cty.DynamicVal)
		h.PostApply(addr, states.CurrentGen, cty.DynamicVal, nil)
	}

	expected := &countHook{pending: make(map[string]plans.Action)}
	expected.Added = 0
	expected.Changed = 0
	expected.Removed = 4

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected:\n%#v\nGot:\n%#v\n", expected, h)
	}
}

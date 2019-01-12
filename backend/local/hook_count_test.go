package local

import (
	"reflect"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
)

func TestCountHook_impl(t *testing.T) {
	var _ terraform.Hook = new(CountHook)
}

func TestCountHookPostDiff_DestroyDeposed(t *testing.T) {
	h := new(CountHook)

	resources := map[string]*terraform.InstanceDiff{
		"lorem": &terraform.InstanceDiff{DestroyDeposed: true},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PostDiff(addr, states.DeposedKey("deadbeef"), plans.Delete, cty.DynamicVal, cty.DynamicVal)
	}

	expected := new(CountHook)
	expected.ToAdd = 0
	expected.ToChange = 0
	expected.ToRemoveAndAdd = 0
	expected.ToRemove = 1

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected %#v, got %#v instead.", expected, h)
	}
}

func TestCountHookPostDiff_DestroyOnly(t *testing.T) {
	h := new(CountHook)

	resources := map[string]*terraform.InstanceDiff{
		"foo":   &terraform.InstanceDiff{Destroy: true},
		"bar":   &terraform.InstanceDiff{Destroy: true},
		"lorem": &terraform.InstanceDiff{Destroy: true},
		"ipsum": &terraform.InstanceDiff{Destroy: true},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PostDiff(addr, states.CurrentGen, plans.Delete, cty.DynamicVal, cty.DynamicVal)
	}

	expected := new(CountHook)
	expected.ToAdd = 0
	expected.ToChange = 0
	expected.ToRemoveAndAdd = 0
	expected.ToRemove = 4

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected %#v, got %#v instead.", expected, h)
	}
}

func TestCountHookPostDiff_AddOnly(t *testing.T) {
	h := new(CountHook)

	resources := map[string]*terraform.InstanceDiff{
		"foo": &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": &terraform.ResourceAttrDiff{RequiresNew: true},
			},
		},
		"bar": &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": &terraform.ResourceAttrDiff{RequiresNew: true},
			},
		},
		"lorem": &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": &terraform.ResourceAttrDiff{RequiresNew: true},
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

	expected := new(CountHook)
	expected.ToAdd = 3
	expected.ToChange = 0
	expected.ToRemoveAndAdd = 0
	expected.ToRemove = 0

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected %#v, got %#v instead.", expected, h)
	}
}

func TestCountHookPostDiff_ChangeOnly(t *testing.T) {
	h := new(CountHook)

	resources := map[string]*terraform.InstanceDiff{
		"foo": &terraform.InstanceDiff{
			Destroy: false,
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": &terraform.ResourceAttrDiff{},
			},
		},
		"bar": &terraform.InstanceDiff{
			Destroy: false,
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": &terraform.ResourceAttrDiff{},
			},
		},
		"lorem": &terraform.InstanceDiff{
			Destroy: false,
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": &terraform.ResourceAttrDiff{},
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

	expected := new(CountHook)
	expected.ToAdd = 0
	expected.ToChange = 3
	expected.ToRemoveAndAdd = 0
	expected.ToRemove = 0

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected %#v, got %#v instead.", expected, h)
	}
}

func TestCountHookPostDiff_Mixed(t *testing.T) {
	h := new(CountHook)

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

	expected := new(CountHook)
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
	h := new(CountHook)

	resources := map[string]*terraform.InstanceDiff{
		"foo":   &terraform.InstanceDiff{},
		"bar":   &terraform.InstanceDiff{},
		"lorem": &terraform.InstanceDiff{},
		"ipsum": &terraform.InstanceDiff{},
	}

	for k := range resources {
		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: k,
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		h.PostDiff(addr, states.CurrentGen, plans.NoOp, cty.DynamicVal, cty.DynamicVal)
	}

	expected := new(CountHook)
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
	h := new(CountHook)

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

	expected := new(CountHook)
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
	h := new(CountHook)

	resources := map[string]*terraform.InstanceDiff{
		"foo": &terraform.InstanceDiff{
			Destroy: false,
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": &terraform.ResourceAttrDiff{},
			},
		},
		"bar": &terraform.InstanceDiff{
			Destroy: false,
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": &terraform.ResourceAttrDiff{},
			},
		},
		"lorem": &terraform.InstanceDiff{
			Destroy: false,
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": &terraform.ResourceAttrDiff{},
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

	expected := &CountHook{pending: make(map[string]plans.Action)}
	expected.Added = 0
	expected.Changed = 3
	expected.Removed = 0

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected:\n%#v\nGot:\n%#v\n", expected, h)
	}
}

func TestCountHookApply_DestroyOnly(t *testing.T) {
	h := new(CountHook)

	resources := map[string]*terraform.InstanceDiff{
		"foo":   &terraform.InstanceDiff{Destroy: true},
		"bar":   &terraform.InstanceDiff{Destroy: true},
		"lorem": &terraform.InstanceDiff{Destroy: true},
		"ipsum": &terraform.InstanceDiff{Destroy: true},
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

	expected := &CountHook{pending: make(map[string]plans.Action)}
	expected.Added = 0
	expected.Changed = 0
	expected.Removed = 4

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected:\n%#v\nGot:\n%#v\n", expected, h)
	}
}

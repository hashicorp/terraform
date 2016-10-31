package command

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestCountHook_impl(t *testing.T) {
	var _ terraform.Hook = new(CountHook)
}

func TestCountHookPostDiff_DestroyOnly(t *testing.T) {
	h := new(CountHook)

	resources := map[string]*terraform.InstanceDiff{
		"foo":   {Destroy: true},
		"bar":   {Destroy: true},
		"lorem": {Destroy: true},
		"ipsum": {Destroy: true},
	}

	n := &terraform.InstanceInfo{} // TODO

	for _, d := range resources {
		h.PostDiff(n, d)
	}

	expected := new(CountHook)
	expected.ToAdd = 0
	expected.ToChange = 0
	expected.ToRemoveAndAdd = 0
	expected.ToRemove = 4

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected %#v, got %#v instead.",
			expected, h)
	}
}

func TestCountHookPostDiff_AddOnly(t *testing.T) {
	h := new(CountHook)

	resources := map[string]*terraform.InstanceDiff{
		"foo": {
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": {RequiresNew: true},
			},
		},
		"bar": {
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": {RequiresNew: true},
			},
		},
		"lorem": {
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": {RequiresNew: true},
			},
		},
	}

	n := &terraform.InstanceInfo{}

	for _, d := range resources {
		h.PostDiff(n, d)
	}

	expected := new(CountHook)
	expected.ToAdd = 3
	expected.ToChange = 0
	expected.ToRemoveAndAdd = 0
	expected.ToRemove = 0

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected %#v, got %#v instead.",
			expected, h)
	}
}

func TestCountHookPostDiff_ChangeOnly(t *testing.T) {
	h := new(CountHook)

	resources := map[string]*terraform.InstanceDiff{
		"foo": {
			Destroy: false,
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": {},
			},
		},
		"bar": {
			Destroy: false,
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": {},
			},
		},
		"lorem": {
			Destroy: false,
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": {},
			},
		},
	}

	n := &terraform.InstanceInfo{}

	for _, d := range resources {
		h.PostDiff(n, d)
	}

	expected := new(CountHook)
	expected.ToAdd = 0
	expected.ToChange = 3
	expected.ToRemoveAndAdd = 0
	expected.ToRemove = 0

	if !reflect.DeepEqual(expected, h) {
		t.Fatalf("Expected %#v, got %#v instead.",
			expected, h)
	}
}

func TestCountHookPostDiff_Mixed(t *testing.T) {
	h := new(CountHook)

	resources := map[string]*terraform.InstanceDiff{
		"foo": {
			Destroy: true,
		},
		"bar": {},
		"lorem": {
			Destroy: false,
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": {},
			},
		},
		"ipsum": {Destroy: true},
	}

	n := &terraform.InstanceInfo{}

	for _, d := range resources {
		h.PostDiff(n, d)
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
		"foo":   {},
		"bar":   {},
		"lorem": {},
		"ipsum": {},
	}

	n := &terraform.InstanceInfo{}

	for _, d := range resources {
		h.PostDiff(n, d)
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

	resources := map[string]*terraform.InstanceDiff{
		"data.foo": {
			Destroy: true,
		},
		"data.bar": {},
		"data.lorem": {
			Destroy: false,
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": {},
			},
		},
		"data.ipsum": {Destroy: true},
	}

	for k, d := range resources {
		n := &terraform.InstanceInfo{Id: k}
		h.PostDiff(n, d)
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

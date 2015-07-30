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
		"foo":   &terraform.InstanceDiff{Destroy: true},
		"bar":   &terraform.InstanceDiff{Destroy: true},
		"lorem": &terraform.InstanceDiff{Destroy: true},
		"ipsum": &terraform.InstanceDiff{Destroy: true},
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
		"foo": &terraform.InstanceDiff{
			Destroy: true,
		},
		"bar": &terraform.InstanceDiff{},
		"lorem": &terraform.InstanceDiff{
			Destroy: false,
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"foo": &terraform.ResourceAttrDiff{},
			},
		},
		"ipsum": &terraform.InstanceDiff{Destroy: true},
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
		"foo":   &terraform.InstanceDiff{},
		"bar":   &terraform.InstanceDiff{},
		"lorem": &terraform.InstanceDiff{},
		"ipsum": &terraform.InstanceDiff{},
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

package terraform

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
)

func TestInstanceState_MergeDiff(t *testing.T) {
	is := InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo":  "bar",
			"port": "8000",
		},
	}

	diff := &InstanceDiff{
		Attributes: map[string]*ResourceAttrDiff{
			"foo": &ResourceAttrDiff{
				Old: "bar",
				New: "baz",
			},
			"bar": &ResourceAttrDiff{
				Old: "",
				New: "foo",
			},
			"baz": &ResourceAttrDiff{
				Old:         "",
				New:         "foo",
				NewComputed: true,
			},
			"port": &ResourceAttrDiff{
				NewRemoved: true,
			},
		},
	}

	is2 := is.MergeDiff(diff)

	expected := map[string]string{
		"foo": "baz",
		"bar": "foo",
		"baz": config.UnknownVariableValue,
	}

	if !reflect.DeepEqual(expected, is2.Attributes) {
		t.Fatalf("bad: %#v", is2.Attributes)
	}
}

func TestInstanceState_MergeDiff_nil(t *testing.T) {
	var is *InstanceState = nil

	diff := &InstanceDiff{
		Attributes: map[string]*ResourceAttrDiff{
			"foo": &ResourceAttrDiff{
				Old: "",
				New: "baz",
			},
		},
	}

	is2 := is.MergeDiff(diff)

	expected := map[string]string{
		"foo": "baz",
	}

	if !reflect.DeepEqual(expected, is2.Attributes) {
		t.Fatalf("bad: %#v", is2.Attributes)
	}
}

func TestInstanceState_MergeDiff_nilDiff(t *testing.T) {
	is := InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo": "bar",
		},
	}

	is2 := is.MergeDiff(nil)

	expected := map[string]string{
		"foo": "bar",
	}

	if !reflect.DeepEqual(expected, is2.Attributes) {
		t.Fatalf("bad: %#v", is2.Attributes)
	}
}

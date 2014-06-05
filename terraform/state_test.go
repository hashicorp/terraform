package terraform

import (
	"reflect"
	"testing"
)

func TestResourceState_MergeDiff(t *testing.T) {
	rs := ResourceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo": "bar",
		},
	}

	diff := map[string]*ResourceAttrDiff{
		"foo": &ResourceAttrDiff{
			Old: "bar",
			New: "baz",
		},
		"bar": &ResourceAttrDiff{
			Old: "",
			New: "foo",
		},
	}

	rs2 := rs.MergeDiff(diff)

	expected := map[string]string{
		"foo": "baz",
		"bar": "foo",
	}

	if !reflect.DeepEqual(expected, rs2.Attributes) {
		t.Fatalf("bad: %#v", rs2.Attributes)
	}
}

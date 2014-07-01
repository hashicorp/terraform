package terraform

import (
	"reflect"
	"testing"
)

func TestResource_Vars(t *testing.T) {
	r := new(Resource)

	if len(r.Vars()) > 0 {
		t.Fatalf("bad: %#v", r.Vars())
	}

	r = &Resource{
		Id: "key",
		State: &ResourceState{
			Attributes: map[string]string{
				"foo": "bar",
			},
		},
	}

	expected := map[string]string{
		"key.foo": "bar",
	}
	actual := r.Vars()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

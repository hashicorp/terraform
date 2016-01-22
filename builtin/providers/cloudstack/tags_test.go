package cloudstack

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDiffTags(t *testing.T) {
	cases := []struct {
		Old, New       map[string]interface{}
		Create, Remove map[string]string
	}{
		// Basic add/remove
		{
			Old: map[string]interface{}{
				"foo": "bar",
			},
			New: map[string]interface{}{
				"bar": "baz",
			},
			Create: map[string]string{
				"bar": "baz",
			},
			Remove: map[string]string{
				"foo": "bar",
			},
		},

		// Modify
		{
			Old: map[string]interface{}{
				"foo": "bar",
			},
			New: map[string]interface{}{
				"foo": "baz",
			},
			Create: map[string]string{
				"foo": "baz",
			},
			Remove: map[string]string{
				"foo": "bar",
			},
		},
	}

	for i, tc := range cases {
		r, c := diffTags(tagsFromSchema(tc.Old), tagsFromSchema(tc.New))
		if !reflect.DeepEqual(r, tc.Remove) {
			t.Fatalf("%d: bad remove: %#v", i, r)
		}
		if !reflect.DeepEqual(c, tc.Create) {
			t.Fatalf("%d: bad create: %#v", i, c)
		}
	}
}

// testAccCheckTags can be used to check the tags on a resource.
func testAccCheckTags(tags map[string]string, key string, value string) error {
	v, ok := tags[key]
	if !ok {
		return fmt.Errorf("Missing tag: %s", key)
	}

	if v != value {
		return fmt.Errorf("%s: bad value: %s", key, v)
	}

	return nil
}

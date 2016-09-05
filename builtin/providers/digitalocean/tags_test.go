package digitalocean

import (
	"reflect"
	"testing"
)

func TestDiffTags(t *testing.T) {
	cases := []struct {
		Old, New       []interface{}
		Create, Remove map[string]string
	}{
		// Basic add/remove
		{
			Old: []interface{}{
				"foo",
			},
			New: []interface{}{
				"bar",
			},
			Create: map[string]string{
				"bar": "bar",
			},
			Remove: map[string]string{
				"foo": "foo",
			},
		},

		// Noop
		{
			Old: []interface{}{
				"foo",
			},
			New: []interface{}{
				"foo",
			},
			Create: map[string]string{},
			Remove: map[string]string{},
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

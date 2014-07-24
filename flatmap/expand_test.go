package flatmap

import (
	"reflect"
	"testing"
)

func TestExpand(t *testing.T) {
	cases := []struct {
		Map    map[string]string
		Key    string
		Output interface{}
	}{
		{
			Map: map[string]string{
				"foo": "bar",
				"bar": "baz",
			},
			Key:    "foo",
			Output: "bar",
		},

		{
			Map: map[string]string{
				"foo.#": "2",
				"foo.0": "one",
				"foo.1": "two",
			},
			Key: "foo",
			Output: []interface{}{
				"one",
				"two",
			},
		},

		{
			Map: map[string]string{
				"foo.#":         "1",
				"foo.0.name":    "bar",
				"foo.0.port":    "3000",
				"foo.0.enabled": "true",
			},
			Key: "foo",
			Output: []interface{}{
				map[string]interface{}{
					"name":    "bar",
					"port":    "3000",
					"enabled": true,
				},
			},
		},

		{
			Map: map[string]string{
				"foo.#":         "1",
				"foo.0.name":    "bar",
				"foo.0.ports.#": "2",
				"foo.0.ports.0": "1",
				"foo.0.ports.1": "2",
			},
			Key: "foo",
			Output: []interface{}{
				map[string]interface{}{
					"name": "bar",
					"ports": []interface{}{
						"1",
						"2",
					},
				},
			},
		},
	}

	for _, tc := range cases {
		actual := Expand(tc.Map, tc.Key)
		if !reflect.DeepEqual(actual, tc.Output) {
			t.Errorf(
				"Key: %v\nMap:\n\n%#v\n\nOutput:\n\n%#v\n\nExpected:\n\n%#v\n",
				tc.Key,
				tc.Map,
				actual,
				tc.Output)
		}
	}
}

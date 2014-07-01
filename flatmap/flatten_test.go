package flatmap

import (
	"reflect"
	"testing"
)

func TestFlatten(t *testing.T) {
	cases := []struct {
		Input  map[string]interface{}
		Output map[string]string
	}{
		{
			Input: map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
			},
			Output: map[string]string{
				"foo": "bar",
				"bar": "baz",
			},
		},

		{
			Input: map[string]interface{}{
				"foo": []string{
					"one",
					"two",
				},
			},
			Output: map[string]string{
				"foo.#": "2",
				"foo.0": "one",
				"foo.1": "two",
			},
		},

		{
			Input: map[string]interface{}{
				"foo": []map[interface{}]interface{}{
					map[interface{}]interface{}{
						"name":    "bar",
						"port":    3000,
						"enabled": true,
					},
				},
			},
			Output: map[string]string{
				"foo.#":         "1",
				"foo.0.name":    "bar",
				"foo.0.port":    "3000",
				"foo.0.enabled": "true",
			},
		},
	}

	for _, tc := range cases {
		actual := Flatten(tc.Input)
		if !reflect.DeepEqual(actual, tc.Output) {
			t.Fatalf(
				"Input:\n\n%#v\n\nOutput:\n\n%#v\n\nExpected:\n\n%#v\n",
				tc.Input,
				actual,
				tc.Output)
		}
	}
}

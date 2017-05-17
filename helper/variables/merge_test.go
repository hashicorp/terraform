package variables

import (
	"fmt"
	"reflect"
	"testing"
)

func TestMerge(t *testing.T) {
	cases := []struct {
		Name     string
		A, B     map[string]interface{}
		Expected map[string]interface{}
	}{
		{
			"basic key/value",
			map[string]interface{}{
				"foo": "bar",
			},
			map[string]interface{}{
				"bar": "baz",
			},
			map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
			},
		},

		{
			"map unset",
			map[string]interface{}{
				"foo": "bar",
			},
			map[string]interface{}{
				"bar": map[string]interface{}{
					"foo": "bar",
				},
			},
			map[string]interface{}{
				"foo": "bar",
				"bar": map[string]interface{}{
					"foo": "bar",
				},
			},
		},

		{
			"map merge",
			map[string]interface{}{
				"foo": "bar",
				"bar": map[string]interface{}{
					"bar": "baz",
				},
			},
			map[string]interface{}{
				"bar": map[string]interface{}{
					"foo": "bar",
				},
			},
			map[string]interface{}{
				"foo": "bar",
				"bar": map[string]interface{}{
					"foo": "bar",
					"bar": "baz",
				},
			},
		},

		{
			"basic k/v with lists",
			map[string]interface{}{
				"foo": "bar",
				"bar": []interface{}{"foo"},
			},
			map[string]interface{}{
				"bar": []interface{}{"bar"},
			},
			map[string]interface{}{
				"foo": "bar",
				"bar": []interface{}{"bar"},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			actual := Merge(tc.A, tc.B)
			if !reflect.DeepEqual(tc.Expected, actual) {
				t.Fatalf("bad: %#v", actual)
			}
		})
	}
}

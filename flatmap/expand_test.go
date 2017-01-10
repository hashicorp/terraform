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

		{
			Map: map[string]string{
				"list_of_map.#":   "2",
				"list_of_map.0.%": "1",
				"list_of_map.0.a": "1",
				"list_of_map.1.%": "2",
				"list_of_map.1.b": "2",
				"list_of_map.1.c": "3",
			},
			Key: "list_of_map",
			Output: []interface{}{
				map[string]interface{}{
					"a": "1",
				},
				map[string]interface{}{
					"b": "2",
					"c": "3",
				},
			},
		},

		{
			Map: map[string]string{
				"map_of_list.%":       "2",
				"map_of_list.list2.#": "1",
				"map_of_list.list2.0": "c",
				"map_of_list.list1.#": "2",
				"map_of_list.list1.0": "a",
				"map_of_list.list1.1": "b",
			},
			Key: "map_of_list",
			Output: map[string]interface{}{
				"list1": []interface{}{"a", "b"},
				"list2": []interface{}{"c"},
			},
		},

		{
			Map: map[string]string{
				"set.#":    "3",
				"set.1234": "a",
				"set.1235": "b",
				"set.1236": "c",
			},
			Key:    "set",
			Output: []interface{}{"a", "b", "c"},
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

package schema

import (
	"reflect"
	"testing"
)

func TestMapFieldWriter_impl(t *testing.T) {
	var _ FieldWriter = new(MapFieldWriter)
}

func TestMapFieldWriter(t *testing.T) {
	schema := map[string]*Schema{
		"bool":   &Schema{Type: TypeBool},
		"int":    &Schema{Type: TypeInt},
		"string": &Schema{Type: TypeString},
		"list": &Schema{
			Type: TypeList,
			Elem: &Schema{Type: TypeString},
		},
		"listInt": &Schema{
			Type: TypeList,
			Elem: &Schema{Type: TypeInt},
		},
		"listResource": &Schema{
			Type:     TypeList,
			Optional: true,
			Computed: true,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"value": &Schema{
						Type:     TypeInt,
						Optional: true,
					},
				},
			},
		},
		"map": &Schema{Type: TypeMap},
		"set": &Schema{
			Type: TypeSet,
			Elem: &Schema{Type: TypeInt},
			Set: func(a interface{}) int {
				return a.(int)
			},
		},
		"setDeep": &Schema{
			Type: TypeSet,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"index": &Schema{Type: TypeInt},
					"value": &Schema{Type: TypeString},
				},
			},
			Set: func(a interface{}) int {
				return a.(map[string]interface{})["index"].(int)
			},
		},
	}

	cases := map[string]struct {
		Addr  []string
		Value interface{}
		Err   bool
		Out   map[string]string
	}{
		"noexist": {
			[]string{"noexist"},
			42,
			true,
			map[string]string{},
		},

		"bool": {
			[]string{"bool"},
			false,
			false,
			map[string]string{
				"bool": "false",
			},
		},

		"int": {
			[]string{"int"},
			42,
			false,
			map[string]string{
				"int": "42",
			},
		},

		"string": {
			[]string{"string"},
			"42",
			false,
			map[string]string{
				"string": "42",
			},
		},

		"string nil": {
			[]string{"string"},
			nil,
			false,
			map[string]string{
				"string": "",
			},
		},

		"list of resources": {
			[]string{"listResource"},
			[]interface{}{
				map[string]interface{}{
					"value": 80,
				},
			},
			false,
			map[string]string{
				"listResource.#":       "1",
				"listResource.0.value": "80",
			},
		},

		"list of resources empty": {
			[]string{"listResource"},
			[]interface{}{},
			false,
			map[string]string{
				"listResource.#": "0",
			},
		},

		"list of resources nil": {
			[]string{"listResource"},
			nil,
			false,
			map[string]string{
				"listResource.#": "0",
			},
		},

		"list of strings": {
			[]string{"list"},
			[]interface{}{"foo", "bar"},
			false,
			map[string]string{
				"list.#": "2",
				"list.0": "foo",
				"list.1": "bar",
			},
		},

		"list element": {
			[]string{"list", "0"},
			"string",
			true,
			map[string]string{},
		},

		"map": {
			[]string{"map"},
			map[string]interface{}{"foo": "bar"},
			false,
			map[string]string{
				"map.%":   "1",
				"map.foo": "bar",
			},
		},

		"map delete": {
			[]string{"map"},
			nil,
			false,
			map[string]string{
				"map": "",
			},
		},

		"map element": {
			[]string{"map", "foo"},
			"bar",
			true,
			map[string]string{},
		},

		"set": {
			[]string{"set"},
			[]interface{}{1, 2, 5},
			false,
			map[string]string{
				"set.#": "3",
				"set.1": "1",
				"set.2": "2",
				"set.5": "5",
			},
		},

		"set nil": {
			[]string{"set"},
			nil,
			false,
			map[string]string{
				"set.#": "0",
			},
		},

		"set typed nil": {
			[]string{"set"},
			func() *Set { return nil }(),
			false,
			map[string]string{
				"set.#": "0",
			},
		},

		"set resource": {
			[]string{"setDeep"},
			[]interface{}{
				map[string]interface{}{
					"index": 10,
					"value": "foo",
				},
				map[string]interface{}{
					"index": 50,
					"value": "bar",
				},
			},
			false,
			map[string]string{
				"setDeep.#":        "2",
				"setDeep.10.index": "10",
				"setDeep.10.value": "foo",
				"setDeep.50.index": "50",
				"setDeep.50.value": "bar",
			},
		},

		"set element": {
			[]string{"set", "5"},
			5,
			true,
			map[string]string{},
		},

		"full object": {
			nil,
			map[string]interface{}{
				"string": "foo",
				"list":   []interface{}{"foo", "bar"},
			},
			false,
			map[string]string{
				"string": "foo",
				"list.#": "2",
				"list.0": "foo",
				"list.1": "bar",
			},
		},
	}

	for name, tc := range cases {
		w := &MapFieldWriter{Schema: schema}
		err := w.WriteField(tc.Addr, tc.Value)
		if err != nil != tc.Err {
			t.Fatalf("%s: err: %s", name, err)
		}

		actual := w.Map()
		if !reflect.DeepEqual(actual, tc.Out) {
			t.Fatalf("%s: bad: %#v", name, actual)
		}
	}
}

func TestMapFieldWriterCleanSet(t *testing.T) {
	schema := map[string]*Schema{
		"setDeep": &Schema{
			Type: TypeSet,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"index": &Schema{Type: TypeInt},
					"value": &Schema{Type: TypeString},
				},
			},
			Set: func(a interface{}) int {
				return a.(map[string]interface{})["index"].(int)
			},
		},
	}

	values := []struct {
		Addr  []string
		Value interface{}
		Out   map[string]string
	}{
		{
			[]string{"setDeep"},
			[]interface{}{
				map[string]interface{}{
					"index": 10,
					"value": "foo",
				},
				map[string]interface{}{
					"index": 50,
					"value": "bar",
				},
			},
			map[string]string{
				"setDeep.#":        "2",
				"setDeep.10.index": "10",
				"setDeep.10.value": "foo",
				"setDeep.50.index": "50",
				"setDeep.50.value": "bar",
			},
		},
		{
			[]string{"setDeep"},
			[]interface{}{
				map[string]interface{}{
					"index": 20,
					"value": "baz",
				},
				map[string]interface{}{
					"index": 60,
					"value": "qux",
				},
			},
			map[string]string{
				"setDeep.#":        "2",
				"setDeep.20.index": "20",
				"setDeep.20.value": "baz",
				"setDeep.60.index": "60",
				"setDeep.60.value": "qux",
			},
		},
		{
			[]string{"setDeep"},
			[]interface{}{
				map[string]interface{}{
					"index": 30,
					"value": "one",
				},
				map[string]interface{}{
					"index": 70,
					"value": "two",
				},
			},
			map[string]string{
				"setDeep.#":        "2",
				"setDeep.30.index": "30",
				"setDeep.30.value": "one",
				"setDeep.70.index": "70",
				"setDeep.70.value": "two",
			},
		},
	}

	w := &MapFieldWriter{Schema: schema}

	for n, tc := range values {
		err := w.WriteField(tc.Addr, tc.Value)
		if err != nil {
			t.Fatalf("%d: err: %s", n, err)
		}

		actual := w.Map()
		if !reflect.DeepEqual(actual, tc.Out) {
			t.Fatalf("%d: bad: %#v", n, actual)
		}
	}
}

func TestMapFieldWriterCleanList(t *testing.T) {
	schema := map[string]*Schema{
		"listDeep": &Schema{
			Type: TypeList,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"thing1": &Schema{Type: TypeString},
					"thing2": &Schema{Type: TypeString},
				},
			},
		},
	}

	values := []struct {
		Addr  []string
		Value interface{}
		Out   map[string]string
	}{
		{
			// Base list
			[]string{"listDeep"},
			[]interface{}{
				map[string]interface{}{
					"thing1": "a",
					"thing2": "b",
				},
				map[string]interface{}{
					"thing1": "c",
					"thing2": "d",
				},
				map[string]interface{}{
					"thing1": "e",
					"thing2": "f",
				},
				map[string]interface{}{
					"thing1": "g",
					"thing2": "h",
				},
			},
			map[string]string{
				"listDeep.#":        "4",
				"listDeep.0.thing1": "a",
				"listDeep.0.thing2": "b",
				"listDeep.1.thing1": "c",
				"listDeep.1.thing2": "d",
				"listDeep.2.thing1": "e",
				"listDeep.2.thing2": "f",
				"listDeep.3.thing1": "g",
				"listDeep.3.thing2": "h",
			},
		},
		{
			// Remove an element
			[]string{"listDeep"},
			[]interface{}{
				map[string]interface{}{
					"thing1": "a",
					"thing2": "b",
				},
				map[string]interface{}{
					"thing1": "c",
					"thing2": "d",
				},
				map[string]interface{}{
					"thing1": "e",
					"thing2": "f",
				},
			},
			map[string]string{
				"listDeep.#":        "3",
				"listDeep.0.thing1": "a",
				"listDeep.0.thing2": "b",
				"listDeep.1.thing1": "c",
				"listDeep.1.thing2": "d",
				"listDeep.2.thing1": "e",
				"listDeep.2.thing2": "f",
			},
		},
		{
			// Rewrite with missing keys. This should normally not be necessary, as
			// hopefully the writers are writing zero values as necessary, but for
			// brevity we want to make sure that what exists in the writer is exactly
			// what the last write looked like coming from the provider.
			[]string{"listDeep"},
			[]interface{}{
				map[string]interface{}{
					"thing1": "a",
				},
				map[string]interface{}{
					"thing1": "c",
				},
				map[string]interface{}{
					"thing1": "e",
				},
			},
			map[string]string{
				"listDeep.#":        "3",
				"listDeep.0.thing1": "a",
				"listDeep.1.thing1": "c",
				"listDeep.2.thing1": "e",
			},
		},
	}

	w := &MapFieldWriter{Schema: schema}

	for n, tc := range values {
		err := w.WriteField(tc.Addr, tc.Value)
		if err != nil {
			t.Fatalf("%d: err: %s", n, err)
		}

		actual := w.Map()
		if !reflect.DeepEqual(actual, tc.Out) {
			t.Fatalf("%d: bad: %#v", n, actual)
		}
	}
}

func TestMapFieldWriterCleanMap(t *testing.T) {
	schema := map[string]*Schema{
		"map": &Schema{
			Type: TypeMap,
		},
	}

	values := []struct {
		Value interface{}
		Out   map[string]string
	}{
		{
			// Base map
			map[string]interface{}{
				"thing1": "a",
				"thing2": "b",
				"thing3": "c",
				"thing4": "d",
			},
			map[string]string{
				"map.%":      "4",
				"map.thing1": "a",
				"map.thing2": "b",
				"map.thing3": "c",
				"map.thing4": "d",
			},
		},
		{
			// Base map
			map[string]interface{}{
				"thing1": "a",
				"thing2": "b",
				"thing4": "d",
			},
			map[string]string{
				"map.%":      "3",
				"map.thing1": "a",
				"map.thing2": "b",
				"map.thing4": "d",
			},
		},
	}

	w := &MapFieldWriter{Schema: schema}

	for n, tc := range values {
		err := w.WriteField([]string{"map"}, tc.Value)
		if err != nil {
			t.Fatalf("%d: err: %s", n, err)
		}

		actual := w.Map()
		if !reflect.DeepEqual(actual, tc.Out) {
			t.Fatalf("%d: bad: %#v", n, actual)
		}
	}
}

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

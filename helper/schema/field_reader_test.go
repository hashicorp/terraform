package schema

import (
	"reflect"
	"testing"
)

func TestMapFieldReader_impl(t *testing.T) {
	var _ FieldReader = new(MapFieldReader)
}

func TestMapFieldReader(t *testing.T) {
	r := &MapFieldReader{
		Map: map[string]string{
			"bool":   "true",
			"int":    "42",
			"string": "string",

			"list.#": "2",
			"list.0": "foo",
			"list.1": "bar",

			"listInt.#": "2",
			"listInt.0": "21",
			"listInt.1": "42",

			"map.foo": "bar",
			"map.bar": "baz",

			"set.#":  "2",
			"set.10": "10",
			"set.50": "50",

			"setDeep.#":        "2",
			"setDeep.10.index": "10",
			"setDeep.10.value": "foo",
			"setDeep.50.index": "50",
			"setDeep.50.value": "bar",
		},
	}

	cases := map[string]struct {
		Addr   []string
		Schema *Schema
		Out    interface{}
		OutOk  bool
		OutErr bool
	}{
		"noexist": {
			[]string{"boolNOPE"},
			&Schema{Type: TypeBool},
			nil,
			false,
			false,
		},

		"bool": {
			[]string{"bool"},
			&Schema{Type: TypeBool},
			true,
			true,
			false,
		},

		"int": {
			[]string{"int"},
			&Schema{Type: TypeInt},
			42,
			true,
			false,
		},

		"string": {
			[]string{"string"},
			&Schema{Type: TypeString},
			"string",
			true,
			false,
		},

		"list": {
			[]string{"list"},
			&Schema{
				Type: TypeList,
				Elem: &Schema{Type: TypeString},
			},
			[]interface{}{
				"foo",
				"bar",
			},
			true,
			false,
		},

		"listInt": {
			[]string{"listInt"},
			&Schema{
				Type: TypeList,
				Elem: &Schema{Type: TypeInt},
			},
			[]interface{}{
				21,
				42,
			},
			true,
			false,
		},

		"map": {
			[]string{"map"},
			&Schema{Type: TypeMap},
			map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
			},
			true,
			false,
		},

		"mapelem": {
			[]string{"map", "foo"},
			&Schema{Type: TypeString},
			"bar",
			true,
			false,
		},

		"set": {
			[]string{"set"},
			&Schema{
				Type: TypeSet,
				Elem: &Schema{Type: TypeInt},
				Set: func(a interface{}) int {
					return a.(int)
				},
			},
			[]interface{}{10, 50},
			true,
			false,
		},

		"setDeep": {
			[]string{"setDeep"},
			&Schema{
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
			true,
			false,
		},
	}

	for name, tc := range cases {
		out, outOk, outErr := r.ReadField(tc.Addr, tc.Schema)
		if (outErr != nil) != tc.OutErr {
			t.Fatalf("%s: err: %s", name, outErr)
		}

		if s, ok := out.(*Set); ok {
			// If it is a set, convert to a list so its more easily checked.
			out = s.List()
		}

		if !reflect.DeepEqual(out, tc.Out) {
			t.Fatalf("%s: out: %#v", name, out)
		}
		if outOk != tc.OutOk {
			t.Fatalf("%s: outOk: %#v", name, outOk)
		}
	}
}

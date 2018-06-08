package schema

import (
	"reflect"
	"testing"
)

func TestAddrToSchema(t *testing.T) {
	cases := map[string]struct {
		Addr   []string
		Schema map[string]*Schema
		Result []ValueType
	}{
		"full object": {
			[]string{},
			map[string]*Schema{
				"list": &Schema{
					Type: TypeList,
					Elem: &Schema{Type: TypeInt},
				},
			},
			[]ValueType{typeObject},
		},

		"list": {
			[]string{"list"},
			map[string]*Schema{
				"list": &Schema{
					Type: TypeList,
					Elem: &Schema{Type: TypeInt},
				},
			},
			[]ValueType{TypeList},
		},

		"list.#": {
			[]string{"list", "#"},
			map[string]*Schema{
				"list": &Schema{
					Type: TypeList,
					Elem: &Schema{Type: TypeInt},
				},
			},
			[]ValueType{TypeList, TypeInt},
		},

		"list.0": {
			[]string{"list", "0"},
			map[string]*Schema{
				"list": &Schema{
					Type: TypeList,
					Elem: &Schema{Type: TypeInt},
				},
			},
			[]ValueType{TypeList, TypeInt},
		},

		"list.0 with resource": {
			[]string{"list", "0"},
			map[string]*Schema{
				"list": &Schema{
					Type: TypeList,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"field": &Schema{Type: TypeString},
						},
					},
				},
			},
			[]ValueType{TypeList, typeObject},
		},

		"list.0.field": {
			[]string{"list", "0", "field"},
			map[string]*Schema{
				"list": &Schema{
					Type: TypeList,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"field": &Schema{Type: TypeString},
						},
					},
				},
			},
			[]ValueType{TypeList, typeObject, TypeString},
		},

		"set": {
			[]string{"set"},
			map[string]*Schema{
				"set": &Schema{
					Type: TypeSet,
					Elem: &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},
			[]ValueType{TypeSet},
		},

		"set.#": {
			[]string{"set", "#"},
			map[string]*Schema{
				"set": &Schema{
					Type: TypeSet,
					Elem: &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},
			[]ValueType{TypeSet, TypeInt},
		},

		"set.0": {
			[]string{"set", "0"},
			map[string]*Schema{
				"set": &Schema{
					Type: TypeSet,
					Elem: &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},
			[]ValueType{TypeSet, TypeInt},
		},

		"set.0 with resource": {
			[]string{"set", "0"},
			map[string]*Schema{
				"set": &Schema{
					Type: TypeSet,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"field": &Schema{Type: TypeString},
						},
					},
				},
			},
			[]ValueType{TypeSet, typeObject},
		},

		"mapElem": {
			[]string{"map", "foo"},
			map[string]*Schema{
				"map": &Schema{Type: TypeMap},
			},
			[]ValueType{TypeMap, TypeString},
		},

		"setDeep": {
			[]string{"set", "50", "index"},
			map[string]*Schema{
				"set": &Schema{
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
			},
			[]ValueType{TypeSet, typeObject, TypeInt},
		},
	}

	for name, tc := range cases {
		result := addrToSchema(tc.Addr, tc.Schema)
		types := make([]ValueType, len(result))
		for i, v := range result {
			types[i] = v.Type
		}

		if !reflect.DeepEqual(types, tc.Result) {
			t.Fatalf("%s: %#v", name, types)
		}
	}
}

// testFieldReader is a helper that should be used to verify that
// a FieldReader behaves properly in all the common cases.
func testFieldReader(t *testing.T, f func(map[string]*Schema) FieldReader) {
	schema := map[string]*Schema{
		// Primitives
		"bool":   &Schema{Type: TypeBool},
		"float":  &Schema{Type: TypeFloat},
		"int":    &Schema{Type: TypeInt},
		"string": &Schema{Type: TypeString},

		// Lists
		"list": &Schema{
			Type: TypeList,
			Elem: &Schema{Type: TypeString},
		},
		"listInt": &Schema{
			Type: TypeList,
			Elem: &Schema{Type: TypeInt},
		},
		"listMap": &Schema{
			Type: TypeList,
			Elem: &Schema{
				Type: TypeMap,
			},
		},

		// Maps
		"map": &Schema{Type: TypeMap},
		"mapInt": &Schema{
			Type: TypeMap,
			Elem: TypeInt,
		},

		// This is used to verify that the type of a Map can be specified using the
		// same syntax as for lists (as a nested *Schema passed to Elem)
		"mapIntNestedSchema": &Schema{
			Type: TypeMap,
			Elem: &Schema{Type: TypeInt},
		},
		"mapFloat": &Schema{
			Type: TypeMap,
			Elem: TypeFloat,
		},
		"mapBool": &Schema{
			Type: TypeMap,
			Elem: TypeBool,
		},

		// Sets
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
		"setEmpty": &Schema{
			Type: TypeSet,
			Elem: &Schema{Type: TypeInt},
			Set: func(a interface{}) int {
				return a.(int)
			},
		},
	}

	cases := map[string]struct {
		Addr   []string
		Result FieldReadResult
		Err    bool
	}{
		"noexist": {
			[]string{"boolNOPE"},
			FieldReadResult{
				Value:    nil,
				Exists:   false,
				Computed: false,
			},
			false,
		},

		"bool": {
			[]string{"bool"},
			FieldReadResult{
				Value:    true,
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"float": {
			[]string{"float"},
			FieldReadResult{
				Value:    3.1415,
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"int": {
			[]string{"int"},
			FieldReadResult{
				Value:    42,
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"string": {
			[]string{"string"},
			FieldReadResult{
				Value:    "string",
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"list": {
			[]string{"list"},
			FieldReadResult{
				Value: []interface{}{
					"foo",
					"bar",
				},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"listInt": {
			[]string{"listInt"},
			FieldReadResult{
				Value: []interface{}{
					21,
					42,
				},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"map": {
			[]string{"map"},
			FieldReadResult{
				Value: map[string]interface{}{
					"foo": "bar",
					"bar": "baz",
				},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"mapInt": {
			[]string{"mapInt"},
			FieldReadResult{
				Value: map[string]interface{}{
					"one": 1,
					"two": 2,
				},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"mapIntNestedSchema": {
			[]string{"mapIntNestedSchema"},
			FieldReadResult{
				Value: map[string]interface{}{
					"one": 1,
					"two": 2,
				},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"mapFloat": {
			[]string{"mapFloat"},
			FieldReadResult{
				Value: map[string]interface{}{
					"oneDotTwo": 1.2,
				},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"mapBool": {
			[]string{"mapBool"},
			FieldReadResult{
				Value: map[string]interface{}{
					"True":  true,
					"False": false,
				},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"mapelem": {
			[]string{"map", "foo"},
			FieldReadResult{
				Value:    "bar",
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"set": {
			[]string{"set"},
			FieldReadResult{
				Value:    []interface{}{10, 50},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"setDeep": {
			[]string{"setDeep"},
			FieldReadResult{
				Value: []interface{}{
					map[string]interface{}{
						"index": 10,
						"value": "foo",
					},
					map[string]interface{}{
						"index": 50,
						"value": "bar",
					},
				},
				Exists:   true,
				Computed: false,
			},
			false,
		},

		"setEmpty": {
			[]string{"setEmpty"},
			FieldReadResult{
				Value:  []interface{}{},
				Exists: false,
			},
			false,
		},
	}

	for name, tc := range cases {
		r := f(schema)
		out, err := r.ReadField(tc.Addr)
		if err != nil != tc.Err {
			t.Fatalf("%s: err: %s", name, err)
		}
		if s, ok := out.Value.(*Set); ok {
			// If it is a set, convert to a list so its more easily checked.
			out.Value = s.List()
		}
		if !reflect.DeepEqual(tc.Result, out) {
			t.Fatalf("%s: bad: %#v", name, out)
		}
	}
}

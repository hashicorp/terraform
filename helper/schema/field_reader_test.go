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

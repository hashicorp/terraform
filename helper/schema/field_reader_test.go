package schema

import (
	"reflect"
	"testing"
)

func TestAddrToSchema(t *testing.T) {
	cases := map[string]struct {
		Addr   []string
		Schema map[string]*Schema
		Result *Schema
	}{
		"mapElem": {
			[]string{"map", "foo"},
			map[string]*Schema{
				"map": &Schema{Type: TypeMap},
			},
			&Schema{Type: TypeString},
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
			&Schema{Type: TypeInt},
		},
	}

	for name, tc := range cases {
		result := addrToSchema(tc.Addr, tc.Schema)
		if !reflect.DeepEqual(result, tc.Result) {
			t.Fatalf("%s: %#v", name, result)
		}
	}
}

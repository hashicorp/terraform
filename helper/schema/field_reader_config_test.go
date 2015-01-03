package schema

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestConfigFieldReader_impl(t *testing.T) {
	var _ FieldReader = new(ConfigFieldReader)
}

func TestConfigFieldReader(t *testing.T) {
	r := &ConfigFieldReader{
		Schema: map[string]*Schema{
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
		},

		Config: testConfig(t, map[string]interface{}{
			"bool":   true,
			"int":    42,
			"string": "string",

			"list": []interface{}{"foo", "bar"},

			"listInt": []interface{}{21, 42},

			"map": map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
			},

			"set": []interface{}{10, 50},

			"setDeep": []interface{}{
				map[string]interface{}{
					"index": 10,
					"value": "foo",
				},
				map[string]interface{}{
					"index": 50,
					"value": "bar",
				},
			},
		}),
	}

	cases := map[string]struct {
		Addr        []string
		Out         interface{}
		OutOk       bool
		OutComputed bool
		OutErr      bool
	}{
		"noexist": {
			[]string{"boolNOPE"},
			nil,
			false,
			false,
			false,
		},

		"bool": {
			[]string{"bool"},
			true,
			true,
			false,
			false,
		},

		"int": {
			[]string{"int"},
			42,
			true,
			false,
			false,
		},

		"string": {
			[]string{"string"},
			"string",
			true,
			false,
			false,
		},

		"list": {
			[]string{"list"},
			[]interface{}{
				"foo",
				"bar",
			},
			true,
			false,
			false,
		},

		"listInt": {
			[]string{"listInt"},
			[]interface{}{
				21,
				42,
			},
			true,
			false,
			false,
		},

		"map": {
			[]string{"map"},
			map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
			},
			true,
			false,
			false,
		},

		"mapelem": {
			[]string{"map", "foo"},
			"bar",
			true,
			false,
			false,
		},

		"set": {
			[]string{"set"},
			[]interface{}{10, 50},
			true,
			false,
			false,
		},

		"setDeep": {
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
			true,
			false,
			false,
		},
	}

	for name, tc := range cases {
		out, err := r.ReadField(tc.Addr)
		if (err != nil) != tc.OutErr {
			t.Fatalf("%s: err: %s", name, err)
		}
		if out.Computed != tc.OutComputed {
			t.Fatalf("%s: err: %#v", name, out.Computed)
		}

		if s, ok := out.Value.(*Set); ok {
			// If it is a set, convert to a list so its more easily checked.
			out.Value = s.List()
		}

		if !reflect.DeepEqual(out.Value, tc.Out) {
			t.Fatalf("%s: out: %#v", name, out.Value)
		}
		if out.Exists != tc.OutOk {
			t.Fatalf("%s: outOk: %#v", name, out.Exists)
		}
	}
}

func testConfig(
	t *testing.T, raw map[string]interface{}) *terraform.ResourceConfig {
	rc, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return terraform.NewResourceConfig(rc)
}

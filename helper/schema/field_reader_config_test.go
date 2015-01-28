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
	testFieldReader(t, func(s map[string]*Schema) FieldReader {
		return &ConfigFieldReader{
			Schema: s,

			Config: testConfig(t, map[string]interface{}{
				"bool":   true,
				"float":  3.1415,
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
	})
}

func testConfig(
	t *testing.T, raw map[string]interface{}) *terraform.ResourceConfig {
	rc, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return terraform.NewResourceConfig(rc)
}

func TestConfigFieldReader_DefaultHandling(t *testing.T) {
	schema := map[string]*Schema{
		"strWithDefault": &Schema{
			Type:    TypeString,
			Default: "ImADefault",
		},
		"strWithDefaultFunc": &Schema{
			Type: TypeString,
			DefaultFunc: func() (interface{}, error) {
				return "FuncDefault", nil
			},
		},
	}

	cases := map[string]struct {
		Addr   []string
		Result FieldReadResult
		Config *terraform.ResourceConfig
		Err    bool
	}{
		"gets default value when no config set": {
			[]string{"strWithDefault"},
			FieldReadResult{
				Value:    "ImADefault",
				Exists:   true,
				Computed: false,
			},
			testConfig(t, map[string]interface{}{}),
			false,
		},
		"config overrides default value": {
			[]string{"strWithDefault"},
			FieldReadResult{
				Value:    "fromConfig",
				Exists:   true,
				Computed: false,
			},
			testConfig(t, map[string]interface{}{
				"strWithDefault": "fromConfig",
			}),
			false,
		},
		"gets default from function when no config set": {
			[]string{"strWithDefaultFunc"},
			FieldReadResult{
				Value:    "FuncDefault",
				Exists:   true,
				Computed: false,
			},
			testConfig(t, map[string]interface{}{}),
			false,
		},
		"config overrides default function": {
			[]string{"strWithDefaultFunc"},
			FieldReadResult{
				Value:    "fromConfig",
				Exists:   true,
				Computed: false,
			},
			testConfig(t, map[string]interface{}{
				"strWithDefaultFunc": "fromConfig",
			}),
			false,
		},
	}

	for name, tc := range cases {
		r := &ConfigFieldReader{
			Schema: schema,
			Config: tc.Config,
		}
		out, err := r.ReadField(tc.Addr)
		if (err != nil) != tc.Err {
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

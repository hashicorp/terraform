package schema

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/lang/ast"
	"github.com/hashicorp/terraform/helper/hashcode"
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

func TestConfigFieldReader_ComputedMap(t *testing.T) {
	schema := map[string]*Schema{
		"map": &Schema{
			Type:     TypeMap,
			Computed: true,
		},
	}

	cases := map[string]struct {
		Addr   []string
		Result FieldReadResult
		Config *terraform.ResourceConfig
		Err    bool
	}{
		"set, normal": {
			[]string{"map"},
			FieldReadResult{
				Value: map[string]interface{}{
					"foo": "bar",
				},
				Exists:   true,
				Computed: false,
			},
			testConfig(t, map[string]interface{}{
				"map": map[string]interface{}{
					"foo": "bar",
				},
			}),
			false,
		},

		"computed element": {
			[]string{"map"},
			FieldReadResult{
				Exists:   true,
				Computed: true,
			},
			testConfigInterpolate(t, map[string]interface{}{
				"map": map[string]interface{}{
					"foo": "${var.foo}",
				},
			}, map[string]ast.Variable{
				"var.foo": ast.Variable{
					Value: config.UnknownVariableValue,
					Type:  ast.TypeString,
				},
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
		if err != nil != tc.Err {
			t.Fatalf("%s: err: %s", name, err)
		}
		if s, ok := out.Value.(*Set); ok {
			// If it is a set, convert to the raw map
			out.Value = s.m
			if len(s.m) == 0 {
				out.Value = nil
			}
		}
		if !reflect.DeepEqual(tc.Result, out) {
			t.Fatalf("%s: bad: %#v", name, out)
		}
	}
}

func TestConfigFieldReader_ComputedSet(t *testing.T) {
	schema := map[string]*Schema{
		"strSet": &Schema{
			Type: TypeSet,
			Elem: &Schema{Type: TypeString},
			Set: func(v interface{}) int {
				return hashcode.String(v.(string))
			},
		},
	}

	cases := map[string]struct {
		Addr   []string
		Result FieldReadResult
		Config *terraform.ResourceConfig
		Err    bool
	}{
		"set, normal": {
			[]string{"strSet"},
			FieldReadResult{
				Value: map[string]interface{}{
					"2356372769": "foo",
				},
				Exists:   true,
				Computed: false,
			},
			testConfig(t, map[string]interface{}{
				"strSet": []interface{}{"foo"},
			}),
			false,
		},

		"set, computed element": {
			[]string{"strSet"},
			FieldReadResult{
				Value:    nil,
				Exists:   true,
				Computed: true,
			},
			testConfigInterpolate(t, map[string]interface{}{
				"strSet": []interface{}{"${var.foo}"},
			}, map[string]ast.Variable{
				"var.foo": ast.Variable{
					Value: config.UnknownVariableValue,
					Type:  ast.TypeString,
				},
			}),
			false,
		},

		"set, computed element substring": {
			[]string{"strSet"},
			FieldReadResult{
				Value:    nil,
				Exists:   true,
				Computed: true,
			},
			testConfigInterpolate(t, map[string]interface{}{
				"strSet": []interface{}{"${var.foo}/32"},
			}, map[string]ast.Variable{
				"var.foo": ast.Variable{
					Value: config.UnknownVariableValue,
					Type:  ast.TypeString,
				},
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
		if err != nil != tc.Err {
			t.Fatalf("%s: err: %s", name, err)
		}
		if s, ok := out.Value.(*Set); ok {
			// If it is a set, convert to the raw map
			out.Value = s.m
			if len(s.m) == 0 {
				out.Value = nil
			}
		}
		if !reflect.DeepEqual(tc.Result, out) {
			t.Fatalf("%s: bad: %#v", name, out)
		}
	}
}

func testConfig(
	t *testing.T, raw map[string]interface{}) *terraform.ResourceConfig {
	return testConfigInterpolate(t, raw, nil)
}

func testConfigInterpolate(
	t *testing.T,
	raw map[string]interface{},
	vs map[string]ast.Variable) *terraform.ResourceConfig {
	rc, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if len(vs) > 0 {
		if err := rc.Interpolate(vs); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	return terraform.NewResourceConfig(rc)
}

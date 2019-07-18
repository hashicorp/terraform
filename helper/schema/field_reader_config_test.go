package schema

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/hcl2shim"
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
				"mapInt": map[string]interface{}{
					"one": "1",
					"two": "2",
				},
				"mapIntNestedSchema": map[string]interface{}{
					"one": "1",
					"two": "2",
				},
				"mapFloat": map[string]interface{}{
					"oneDotTwo": "1.2",
				},
				"mapBool": map[string]interface{}{
					"True":  "true",
					"False": "false",
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

// This contains custom table tests for our ConfigFieldReader
func TestConfigFieldReader_custom(t *testing.T) {
	schema := map[string]*Schema{
		"bool": &Schema{
			Type: TypeBool,
		},
	}

	cases := map[string]struct {
		Addr   []string
		Result FieldReadResult
		Config *terraform.ResourceConfig
		Err    bool
	}{
		"basic": {
			[]string{"bool"},
			FieldReadResult{
				Value:  true,
				Exists: true,
			},
			testConfig(t, map[string]interface{}{
				"bool": true,
			}),
			false,
		},

		"computed": {
			[]string{"bool"},
			FieldReadResult{
				Exists:   true,
				Computed: true,
			},
			testConfigInterpolate(t, map[string]interface{}{
				"bool": "${var.foo}",
			}, map[string]ast.Variable{
				"var.foo": ast.Variable{
					Value: hcl2shim.UnknownVariableValue,
					Type:  ast.TypeString,
				},
			}),
			false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
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
		})
	}
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
		"listmap": &Schema{
			Type:     TypeMap,
			Computed: true,
			Elem:     TypeList,
		},
		"maplist": &Schema{
			Type:     TypeList,
			Computed: true,
			Elem:     TypeMap,
		},
	}

	cases := []struct {
		Name   string
		Addr   []string
		Result FieldReadResult
		Config *terraform.ResourceConfig
		Err    bool
	}{
		{
			"set, normal",
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

		{
			"computed element",
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
					Value: hcl2shim.UnknownVariableValue,
					Type:  ast.TypeString,
				},
			}),
			false,
		},

		{
			"native map",
			[]string{"map"},
			FieldReadResult{
				Value: map[string]interface{}{
					"bar": "baz",
					"baz": "bar",
				},
				Exists:   true,
				Computed: false,
			},
			testConfigInterpolate(t, map[string]interface{}{
				"map": "${var.foo}",
			}, map[string]ast.Variable{
				"var.foo": ast.Variable{
					Type: ast.TypeMap,
					Value: map[string]ast.Variable{
						"bar": ast.Variable{
							Type:  ast.TypeString,
							Value: "baz",
						},
						"baz": ast.Variable{
							Type:  ast.TypeString,
							Value: "bar",
						},
					},
				},
			}),
			false,
		},

		{
			"map-from-list-of-maps",
			[]string{"maplist", "0"},
			FieldReadResult{
				Value: map[string]interface{}{
					"key": "bar",
				},
				Exists:   true,
				Computed: false,
			},
			testConfigInterpolate(t, map[string]interface{}{
				"maplist": "${var.foo}",
			}, map[string]ast.Variable{
				"var.foo": ast.Variable{
					Type: ast.TypeList,
					Value: []ast.Variable{
						{
							Type: ast.TypeMap,
							Value: map[string]ast.Variable{
								"key": ast.Variable{
									Type:  ast.TypeString,
									Value: "bar",
								},
							},
						},
					},
				},
			}),
			false,
		},

		{
			"value-from-list-of-maps",
			[]string{"maplist", "0", "key"},
			FieldReadResult{
				Value:    "bar",
				Exists:   true,
				Computed: false,
			},
			testConfigInterpolate(t, map[string]interface{}{
				"maplist": "${var.foo}",
			}, map[string]ast.Variable{
				"var.foo": ast.Variable{
					Type: ast.TypeList,
					Value: []ast.Variable{
						{
							Type: ast.TypeMap,
							Value: map[string]ast.Variable{
								"key": ast.Variable{
									Type:  ast.TypeString,
									Value: "bar",
								},
							},
						},
					},
				},
			}),
			false,
		},

		{
			"list-from-map-of-lists",
			[]string{"listmap", "key"},
			FieldReadResult{
				Value:    []interface{}{"bar"},
				Exists:   true,
				Computed: false,
			},
			testConfigInterpolate(t, map[string]interface{}{
				"listmap": "${var.foo}",
			}, map[string]ast.Variable{
				"var.foo": ast.Variable{
					Type: ast.TypeMap,
					Value: map[string]ast.Variable{
						"key": ast.Variable{
							Type: ast.TypeList,
							Value: []ast.Variable{
								ast.Variable{
									Type:  ast.TypeString,
									Value: "bar",
								},
							},
						},
					},
				},
			}),
			false,
		},

		{
			"value-from-map-of-lists",
			[]string{"listmap", "key", "0"},
			FieldReadResult{
				Value:    "bar",
				Exists:   true,
				Computed: false,
			},
			testConfigInterpolate(t, map[string]interface{}{
				"listmap": "${var.foo}",
			}, map[string]ast.Variable{
				"var.foo": ast.Variable{
					Type: ast.TypeMap,
					Value: map[string]ast.Variable{
						"key": ast.Variable{
							Type: ast.TypeList,
							Value: []ast.Variable{
								ast.Variable{
									Type:  ast.TypeString,
									Value: "bar",
								},
							},
						},
					},
				},
			}),
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			r := &ConfigFieldReader{
				Schema: schema,
				Config: tc.Config,
			}
			out, err := r.ReadField(tc.Addr)
			if err != nil != tc.Err {
				t.Fatal(err)
			}
			if s, ok := out.Value.(*Set); ok {
				// If it is a set, convert to the raw map
				out.Value = s.m
				if len(s.m) == 0 {
					out.Value = nil
				}
			}
			if !reflect.DeepEqual(tc.Result, out) {
				t.Fatalf("\nexpected: %#v\ngot:      %#v", tc.Result, out)
			}
		})
	}
}

func TestConfigFieldReader_ComputedSet(t *testing.T) {
	schema := map[string]*Schema{
		"strSet": &Schema{
			Type: TypeSet,
			Elem: &Schema{Type: TypeString},
			Set:  HashString,
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
					Value: hcl2shim.UnknownVariableValue,
					Type:  ast.TypeUnknown,
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
					Value: hcl2shim.UnknownVariableValue,
					Type:  ast.TypeUnknown,
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

func TestConfigFieldReader_computedComplexSet(t *testing.T) {
	hashfunc := func(v interface{}) int {
		var buf bytes.Buffer
		m := v.(map[string]interface{})
		buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
		buf.WriteString(fmt.Sprintf("%s-", m["vhd_uri"].(string)))
		return hashcode.String(buf.String())
	}

	schema := map[string]*Schema{
		"set": &Schema{
			Type: TypeSet,
			Elem: &Resource{
				Schema: map[string]*Schema{
					"name": {
						Type:     TypeString,
						Required: true,
					},

					"vhd_uri": {
						Type:     TypeString,
						Required: true,
					},
				},
			},
			Set: hashfunc,
		},
	}

	cases := map[string]struct {
		Addr   []string
		Result FieldReadResult
		Config *terraform.ResourceConfig
		Err    bool
	}{
		"set, normal": {
			[]string{"set"},
			FieldReadResult{
				Value: map[string]interface{}{
					"532860136": map[string]interface{}{
						"name":    "myosdisk1",
						"vhd_uri": "bar",
					},
				},
				Exists:   true,
				Computed: false,
			},
			testConfig(t, map[string]interface{}{
				"set": []interface{}{
					map[string]interface{}{
						"name":    "myosdisk1",
						"vhd_uri": "bar",
					},
				},
			}),
			false,
		},

		"set, computed element": {
			[]string{"set"},
			FieldReadResult{
				Value: map[string]interface{}{
					"~3596295623": map[string]interface{}{
						"name":    "myosdisk1",
						"vhd_uri": "${var.foo}/bar",
					},
				},
				Exists:   true,
				Computed: false,
			},
			testConfigInterpolate(t, map[string]interface{}{
				"set": []interface{}{
					map[string]interface{}{
						"name":    "myosdisk1",
						"vhd_uri": "${var.foo}/bar",
					},
				},
			}, map[string]ast.Variable{
				"var.foo": ast.Variable{
					Value: hcl2shim.UnknownVariableValue,
					Type:  ast.TypeUnknown,
				},
			}),
			false,
		},

		"set, computed element single": {
			[]string{"set", "~3596295623", "vhd_uri"},
			FieldReadResult{
				Value:    "${var.foo}/bar",
				Exists:   true,
				Computed: true,
			},
			testConfigInterpolate(t, map[string]interface{}{
				"set": []interface{}{
					map[string]interface{}{
						"name":    "myosdisk1",
						"vhd_uri": "${var.foo}/bar",
					},
				},
			}, map[string]ast.Variable{
				"var.foo": ast.Variable{
					Value: hcl2shim.UnknownVariableValue,
					Type:  ast.TypeUnknown,
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

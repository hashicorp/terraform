package terraform

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
)

func TestCoerceMapVariable(t *testing.T) {
	cases := map[string]struct {
		Input      *EvalCoerceMapVariable
		ExpectVars map[string]interface{}
	}{
		"a valid map is untouched": {
			Input: &EvalCoerceMapVariable{
				Variables: map[string]interface{}{
					"amap": map[string]interface{}{"foo": "bar"},
				},
				ModulePath: []string{"root"},
				ModuleTree: testModuleInline(t, map[string]string{
					"main.tf": `
						variable "amap" {
							type = "map"
						}
					`,
				}),
			},
			ExpectVars: map[string]interface{}{
				"amap": map[string]interface{}{"foo": "bar"},
			},
		},
		"a list w/ a single map element is coerced": {
			Input: &EvalCoerceMapVariable{
				Variables: map[string]interface{}{
					"amap": []interface{}{
						map[string]interface{}{"foo": "bar"},
					},
				},
				ModulePath: []string{"root"},
				ModuleTree: testModuleInline(t, map[string]string{
					"main.tf": `
						variable "amap" {
							type = "map"
						}
					`,
				}),
			},
			ExpectVars: map[string]interface{}{
				"amap": map[string]interface{}{"foo": "bar"},
			},
		},
		"a list w/ more than one map element is untouched": {
			Input: &EvalCoerceMapVariable{
				Variables: map[string]interface{}{
					"amap": []interface{}{
						map[string]interface{}{"foo": "bar"},
						map[string]interface{}{"baz": "qux"},
					},
				},
				ModulePath: []string{"root"},
				ModuleTree: testModuleInline(t, map[string]string{
					"main.tf": `
						variable "amap" {
							type = "map"
						}
					`,
				}),
			},
			ExpectVars: map[string]interface{}{
				"amap": []interface{}{
					map[string]interface{}{"foo": "bar"},
					map[string]interface{}{"baz": "qux"},
				},
			},
		},
		"list coercion also works in a module": {
			Input: &EvalCoerceMapVariable{
				Variables: map[string]interface{}{
					"amap": []interface{}{
						map[string]interface{}{"foo": "bar"},
					},
				},
				ModulePath: []string{"root", "middle", "bottom"},
				ModuleTree: testModuleInline(t, map[string]string{
					"top.tf": `
						module "middle" {
							source = "./middle"
						}
					`,
					"middle/mid.tf": `
						module "bottom" {
							source = "./bottom"
							amap {
								foo = "bar"
							}
						}
					`,
					"middle/bottom/bot.tf": `
						variable "amap" {
							type = "map"
						}
					`,
				}),
			},
			ExpectVars: map[string]interface{}{
				"amap": map[string]interface{}{"foo": "bar"},
			},
		},
		"coercion only occurs when target var is a map": {
			Input: &EvalCoerceMapVariable{
				Variables: map[string]interface{}{
					"alist": []interface{}{
						map[string]interface{}{"foo": "bar"},
					},
				},
				ModulePath: []string{"root"},
				ModuleTree: testModuleInline(t, map[string]string{
					"main.tf": `
						variable "alist" {
							type = "list"
						}
					`,
				}),
			},
			ExpectVars: map[string]interface{}{
				"alist": []interface{}{
					map[string]interface{}{"foo": "bar"},
				},
			},
		},
	}

	for tn, tc := range cases {
		_, err := tc.Input.Eval(&MockEvalContext{})
		if err != nil {
			t.Fatalf("%s: Unexpected err: %s", tn, err)
		}
		if !reflect.DeepEqual(tc.Input.Variables, tc.ExpectVars) {
			t.Fatalf("%s: Expected variables:\n\n%#v\n\nGot:\n\n%#v",
				tn, tc.ExpectVars, tc.Input.Variables)
		}
	}
}

func TestEvalVariableBlock(t *testing.T) {
	rc, err := config.NewRawConfig(map[string]interface{}{
		"known":      "foo",
		"known_list": []interface{}{"foo"},
		"known_map": map[string]interface{}{
			"foo": "foo",
		},
		"known_list_of_maps": []map[string]interface{}{
			map[string]interface{}{
				"foo": "foo",
			},
		},
		"computed_map": map[string]interface{}{},
		"computed_list_of_maps": []map[string]interface{}{
			map[string]interface{}{},
		},
		// No computed_list right now, because that isn't currently supported:
		// EvalVariableBlock assumes the final step of the path will always
		// be a map.
	})
	if err != nil {
		t.Fatalf("config.NewRawConfig failed: %s", err)
	}

	cfg := NewResourceConfig(rc)
	cfg.ComputedKeys = []string{
		"computed",
		"computed_map.foo",
		"computed_list_of_maps.0.foo",
	}

	n := &EvalVariableBlock{
		VariableValues: map[string]interface{}{
			// Should be cleared out on Eval
			"should_be_deleted": true,
		},
		Config: &cfg,
	}

	ctx := &MockEvalContext{}
	val, err := n.Eval(ctx)
	if err != nil {
		t.Fatalf("n.Eval failed: %s", err)
	}
	if val != nil {
		t.Fatalf("n.Eval returned non-nil result: %#v", val)
	}

	got := n.VariableValues
	want := map[string]interface{}{
		"known":      "foo",
		"known_list": []interface{}{"foo"},
		"known_map": map[string]interface{}{
			"foo": "foo",
		},
		"known_list_of_maps": []interface{}{
			map[string]interface{}{
				"foo": "foo",
			},
		},
		"computed": config.UnknownVariableValue,
		"computed_map": map[string]interface{}{
			"foo": config.UnknownVariableValue,
		},
		"computed_list_of_maps": []interface{}{
			map[string]interface{}{
				"foo": config.UnknownVariableValue,
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Incorrect variables\ngot:  %#v\nwant: %#v", got, want)
	}
}

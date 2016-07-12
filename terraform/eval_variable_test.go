package terraform

import (
	"reflect"
	"testing"
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

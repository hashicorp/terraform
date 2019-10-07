package lang

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

func TestScopeEvalContext(t *testing.T) {
	data := &dataForTests{
		CountAttrs: map[string]cty.Value{
			"index": cty.NumberIntVal(0),
		},
		ForEachAttrs: map[string]cty.Value{
			"key":   cty.StringVal("a"),
			"value": cty.NumberIntVal(1),
		},
		Resources: map[string]cty.Value{
			"null_resource.foo": cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("bar"),
			}),
			"data.null_data_source.foo": cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("bar"),
			}),
			"null_resource.multi": cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("multi0"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("multi1"),
				}),
			}),
			"null_resource.each": cty.ObjectVal(map[string]cty.Value{
				"each0": cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("each0"),
				}),
				"each1": cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("each1"),
				}),
			}),
			"null_resource.multi[1]": cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("multi1"),
			}),
		},
		LocalValues: map[string]cty.Value{
			"foo": cty.StringVal("bar"),
		},
		Modules: map[string]cty.Value{
			"module.foo": cty.ObjectVal(map[string]cty.Value{
				"output0": cty.StringVal("bar0"),
				"output1": cty.StringVal("bar1"),
			}),
		},
		PathAttrs: map[string]cty.Value{
			"module": cty.StringVal("foo/bar"),
		},
		TerraformAttrs: map[string]cty.Value{
			"workspace": cty.StringVal("default"),
		},
		InputVariables: map[string]cty.Value{
			"baz": cty.StringVal("boop"),
		},
	}

	tests := []struct {
		Expr string
		Want map[string]cty.Value
	}{
		{
			`12`,
			map[string]cty.Value{},
		},
		{
			`count.index`,
			map[string]cty.Value{
				"count": cty.ObjectVal(map[string]cty.Value{
					"index": cty.NumberIntVal(0),
				}),
			},
		},
		{
			`each.key`,
			map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"key": cty.StringVal("a"),
				}),
			},
		},
		{
			`each.value`,
			map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"value": cty.NumberIntVal(1),
				}),
			},
		},
		{
			`local.foo`,
			map[string]cty.Value{
				"local": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("bar"),
				}),
			},
		},
		{
			`null_resource.foo`,
			map[string]cty.Value{
				"null_resource": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("bar"),
					}),
				}),
			},
		},
		{
			`null_resource.foo.attr`,
			map[string]cty.Value{
				"null_resource": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("bar"),
					}),
				}),
			},
		},
		{
			`null_resource.multi`,
			map[string]cty.Value{
				"null_resource": cty.ObjectVal(map[string]cty.Value{
					"multi": cty.TupleVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"attr": cty.StringVal("multi0"),
						}),
						cty.ObjectVal(map[string]cty.Value{
							"attr": cty.StringVal("multi1"),
						}),
					}),
				}),
			},
		},
		{
			// at this level, all instance references return the entire resource
			`null_resource.multi[1]`,
			map[string]cty.Value{
				"null_resource": cty.ObjectVal(map[string]cty.Value{
					"multi": cty.TupleVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"attr": cty.StringVal("multi0"),
						}),
						cty.ObjectVal(map[string]cty.Value{
							"attr": cty.StringVal("multi1"),
						}),
					}),
				}),
			},
		},
		{
			// at this level, all instance references return the entire resource
			`null_resource.each["each1"]`,
			map[string]cty.Value{
				"null_resource": cty.ObjectVal(map[string]cty.Value{
					"each": cty.ObjectVal(map[string]cty.Value{
						"each0": cty.ObjectVal(map[string]cty.Value{
							"attr": cty.StringVal("each0"),
						}),
						"each1": cty.ObjectVal(map[string]cty.Value{
							"attr": cty.StringVal("each1"),
						}),
					}),
				}),
			},
		},
		{
			`foo(null_resource.multi, null_resource.multi[1])`,
			map[string]cty.Value{
				"null_resource": cty.ObjectVal(map[string]cty.Value{
					"multi": cty.TupleVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"attr": cty.StringVal("multi0"),
						}),
						cty.ObjectVal(map[string]cty.Value{
							"attr": cty.StringVal("multi1"),
						}),
					}),
				}),
			},
		},
		{
			`data.null_data_source.foo`,
			map[string]cty.Value{
				"data": cty.ObjectVal(map[string]cty.Value{
					"null_data_source": cty.ObjectVal(map[string]cty.Value{
						"foo": cty.ObjectVal(map[string]cty.Value{
							"attr": cty.StringVal("bar"),
						}),
					}),
				}),
			},
		},
		{
			`module.foo`,
			map[string]cty.Value{
				"module": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.ObjectVal(map[string]cty.Value{
						"output0": cty.StringVal("bar0"),
						"output1": cty.StringVal("bar1"),
					}),
				}),
			},
		},
		{
			`module.foo.output1`,
			map[string]cty.Value{
				"module": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.ObjectVal(map[string]cty.Value{
						"output1": cty.StringVal("bar1"),
					}),
				}),
			},
		},
		{
			`path.module`,
			map[string]cty.Value{
				"path": cty.ObjectVal(map[string]cty.Value{
					"module": cty.StringVal("foo/bar"),
				}),
			},
		},
		{
			`self.baz`,
			map[string]cty.Value{
				"self": cty.ObjectVal(map[string]cty.Value{
					"attr": cty.StringVal("multi1"),
				}),
			},
		},
		{
			`terraform.workspace`,
			map[string]cty.Value{
				"terraform": cty.ObjectVal(map[string]cty.Value{
					"workspace": cty.StringVal("default"),
				}),
			},
		},
		{
			`var.baz`,
			map[string]cty.Value{
				"var": cty.ObjectVal(map[string]cty.Value{
					"baz": cty.StringVal("boop"),
				}),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Expr, func(t *testing.T) {
			expr, parseDiags := hclsyntax.ParseExpression([]byte(test.Expr), "", hcl.Pos{Line: 1, Column: 1})
			if len(parseDiags) != 0 {
				t.Errorf("unexpected diagnostics during parse")
				for _, diag := range parseDiags {
					t.Errorf("- %s", diag)
				}
				return
			}

			refs, refsDiags := ReferencesInExpr(expr)
			if refsDiags.HasErrors() {
				t.Fatal(refsDiags.Err())
			}

			scope := &Scope{
				Data: data,

				// "self" will just be an arbitrary one of the several resource
				// instances we have in our test dataset.
				SelfAddr: addrs.ResourceInstance{
					Resource: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "null_resource",
						Name: "multi",
					},
					Key: addrs.IntKey(1),
				},
			}
			ctx, ctxDiags := scope.EvalContext(refs)
			if ctxDiags.HasErrors() {
				t.Fatal(ctxDiags.Err())
			}

			// For easier test assertions we'll just remove any top-level
			// empty objects from our variables map.
			for k, v := range ctx.Variables {
				if v.RawEquals(cty.EmptyObjectVal) {
					delete(ctx.Variables, k)
				}
			}

			gotVal := cty.ObjectVal(ctx.Variables)
			wantVal := cty.ObjectVal(test.Want)

			if !gotVal.RawEquals(wantVal) {
				// We'll JSON-ize our values here just so it's easier to
				// read them in the assertion output.
				gotJSON := formattedJSONValue(gotVal)
				wantJSON := formattedJSONValue(wantVal)

				t.Errorf(
					"wrong result\nexpr: %s\ngot:  %s\nwant: %s",
					test.Expr, gotJSON, wantJSON,
				)
			}
		})
	}
}

func TestScopeExpandEvalBlock(t *testing.T) {
	nestedObjTy := cty.Object(map[string]cty.Type{
		"boop": cty.String,
	})
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo":         {Type: cty.String, Optional: true},
			"list_of_obj": {Type: cty.List(nestedObjTy), Optional: true},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"bar": {
				Nesting: configschema.NestingMap,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"baz": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
	data := &dataForTests{
		LocalValues: map[string]cty.Value{
			"greeting": cty.StringVal("howdy"),
			"list": cty.ListVal([]cty.Value{
				cty.StringVal("elem0"),
				cty.StringVal("elem1"),
			}),
			"map": cty.MapVal(map[string]cty.Value{
				"key1": cty.StringVal("val1"),
				"key2": cty.StringVal("val2"),
			}),
		},
	}

	tests := map[string]struct {
		Config string
		Want   cty.Value
	}{
		"empty": {
			`
			`,
			cty.ObjectVal(map[string]cty.Value{
				"foo":         cty.NullVal(cty.String),
				"list_of_obj": cty.NullVal(cty.List(nestedObjTy)),
				"bar": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"baz": cty.String,
				})),
			}),
		},
		"literal attribute": {
			`
			foo = "hello"
			`,
			cty.ObjectVal(map[string]cty.Value{
				"foo":         cty.StringVal("hello"),
				"list_of_obj": cty.NullVal(cty.List(nestedObjTy)),
				"bar": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"baz": cty.String,
				})),
			}),
		},
		"variable attribute": {
			`
			foo = local.greeting
			`,
			cty.ObjectVal(map[string]cty.Value{
				"foo":         cty.StringVal("howdy"),
				"list_of_obj": cty.NullVal(cty.List(nestedObjTy)),
				"bar": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"baz": cty.String,
				})),
			}),
		},
		"one static block": {
			`
			bar "static" {}
			`,
			cty.ObjectVal(map[string]cty.Value{
				"foo":         cty.NullVal(cty.String),
				"list_of_obj": cty.NullVal(cty.List(nestedObjTy)),
				"bar": cty.MapVal(map[string]cty.Value{
					"static": cty.ObjectVal(map[string]cty.Value{
						"baz": cty.NullVal(cty.String),
					}),
				}),
			}),
		},
		"two static blocks": {
			`
			bar "static0" {
				baz = 0
			}
			bar "static1" {
				baz = 1
			}
			`,
			cty.ObjectVal(map[string]cty.Value{
				"foo":         cty.NullVal(cty.String),
				"list_of_obj": cty.NullVal(cty.List(nestedObjTy)),
				"bar": cty.MapVal(map[string]cty.Value{
					"static0": cty.ObjectVal(map[string]cty.Value{
						"baz": cty.StringVal("0"),
					}),
					"static1": cty.ObjectVal(map[string]cty.Value{
						"baz": cty.StringVal("1"),
					}),
				}),
			}),
		},
		"dynamic blocks from list": {
			`
			dynamic "bar" {
				for_each = local.list
				labels = [bar.value]
				content {
					baz = bar.key
				}
			}
			`,
			cty.ObjectVal(map[string]cty.Value{
				"foo":         cty.NullVal(cty.String),
				"list_of_obj": cty.NullVal(cty.List(nestedObjTy)),
				"bar": cty.MapVal(map[string]cty.Value{
					"elem0": cty.ObjectVal(map[string]cty.Value{
						"baz": cty.StringVal("0"),
					}),
					"elem1": cty.ObjectVal(map[string]cty.Value{
						"baz": cty.StringVal("1"),
					}),
				}),
			}),
		},
		"dynamic blocks from map": {
			`
			dynamic "bar" {
				for_each = local.map
				labels = [bar.key]
				content {
					baz = bar.value
				}
			}
			`,
			cty.ObjectVal(map[string]cty.Value{
				"foo":         cty.NullVal(cty.String),
				"list_of_obj": cty.NullVal(cty.List(nestedObjTy)),
				"bar": cty.MapVal(map[string]cty.Value{
					"key1": cty.ObjectVal(map[string]cty.Value{
						"baz": cty.StringVal("val1"),
					}),
					"key2": cty.ObjectVal(map[string]cty.Value{
						"baz": cty.StringVal("val2"),
					}),
				}),
			}),
		},
		"list-of-object attribute": {
			`
			list_of_obj = [
				{
					boop = local.greeting
				},
			]
			`,
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
				"list_of_obj": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"boop": cty.StringVal("howdy"),
					}),
				}),
				"bar": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"baz": cty.String,
				})),
			}),
		},
		"list-of-object attribute as blocks": {
			`
			list_of_obj {
				boop = local.greeting
			}
			`,
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
				"list_of_obj": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"boop": cty.StringVal("howdy"),
					}),
				}),
				"bar": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"baz": cty.String,
				})),
			}),
		},
		"lots of things at once": {
			`
			foo = "whoop"
			bar "static0" {
				baz = "s0"
			}
			dynamic "bar" {
				for_each = local.list
				labels = [bar.value]
				content {
					baz = bar.key
				}
			}
			bar "static1" {
				baz = "s1"
			}
			dynamic "bar" {
				for_each = local.map
				labels = [bar.key]
				content {
					baz = bar.value
				}
			}
			bar "static2" {
				baz = "s2"
			}
			`,
			cty.ObjectVal(map[string]cty.Value{
				"foo":         cty.StringVal("whoop"),
				"list_of_obj": cty.NullVal(cty.List(nestedObjTy)),
				"bar": cty.MapVal(map[string]cty.Value{
					"key1": cty.ObjectVal(map[string]cty.Value{
						"baz": cty.StringVal("val1"),
					}),
					"key2": cty.ObjectVal(map[string]cty.Value{
						"baz": cty.StringVal("val2"),
					}),
					"elem0": cty.ObjectVal(map[string]cty.Value{
						"baz": cty.StringVal("0"),
					}),
					"elem1": cty.ObjectVal(map[string]cty.Value{
						"baz": cty.StringVal("1"),
					}),
					"static0": cty.ObjectVal(map[string]cty.Value{
						"baz": cty.StringVal("s0"),
					}),
					"static1": cty.ObjectVal(map[string]cty.Value{
						"baz": cty.StringVal("s1"),
					}),
					"static2": cty.ObjectVal(map[string]cty.Value{
						"baz": cty.StringVal("s2"),
					}),
				}),
			}),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			file, parseDiags := hclsyntax.ParseConfig([]byte(test.Config), "", hcl.Pos{Line: 1, Column: 1})
			if len(parseDiags) != 0 {
				t.Errorf("unexpected diagnostics during parse")
				for _, diag := range parseDiags {
					t.Errorf("- %s", diag)
				}
				return
			}

			body := file.Body
			scope := &Scope{
				Data: data,
			}

			body, expandDiags := scope.ExpandBlock(body, schema)
			if expandDiags.HasErrors() {
				t.Fatal(expandDiags.Err())
			}

			got, valDiags := scope.EvalBlock(body, schema)
			if valDiags.HasErrors() {
				t.Fatal(valDiags.Err())
			}

			if !got.RawEquals(test.Want) {
				// We'll JSON-ize our values here just so it's easier to
				// read them in the assertion output.
				gotJSON := formattedJSONValue(got)
				wantJSON := formattedJSONValue(test.Want)

				t.Errorf(
					"wrong result\nconfig: %s\ngot:   %s\nwant:  %s",
					test.Config, gotJSON, wantJSON,
				)
			}

		})
	}

}

func formattedJSONValue(val cty.Value) string {
	val = cty.UnknownAsNull(val) // since JSON can't represent unknowns
	j, err := ctyjson.Marshal(val, val.Type())
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	json.Indent(&buf, j, "", "  ")
	return buf.String()
}

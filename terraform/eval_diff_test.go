package terraform

import (
	"testing"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func TestProcessIgnoreChangesIndividual(t *testing.T) {
	tests := map[string]struct {
		Old, New cty.Value
		Ignore   []string
		Want     cty.Value
	}{
		"string": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.StringVal("b value"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("new a value"),
				"b": cty.StringVal("new b value"),
			}),
			[]string{"a"},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.StringVal("new b value"),
			}),
		},
		"changed type": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.StringVal("b value"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NumberIntVal(1),
				"b": cty.StringVal("new b value"),
			}),
			[]string{"a"},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a value"),
				"b": cty.StringVal("new b value"),
			}),
		},
		"list": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.StringVal("a0 value"),
					cty.StringVal("a1 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.StringVal("new a0 value"),
					cty.StringVal("new a1 value"),
				}),
				"b": cty.StringVal("new b value"),
			}),
			[]string{"a"},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.StringVal("a0 value"),
					cty.StringVal("a1 value"),
				}),
				"b": cty.StringVal("new b value"),
			}),
		},
		"object attribute": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("a.foo value"),
					"bar": cty.StringVal("a.bar value"),
				}),
				"b": cty.StringVal("b value"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("new a.foo value"),
					"bar": cty.StringVal("new a.bar value"),
				}),
				"b": cty.StringVal("new b value"),
			}),
			[]string{"a.bar"},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ObjectVal(map[string]cty.Value{
					"foo": cty.StringVal("new a.foo value"),
					"bar": cty.StringVal("a.bar value"),
				}),
				"b": cty.StringVal("new b value"),
			}),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ignore := make([]hcl.Traversal, len(test.Ignore))
			for i, ignoreStr := range test.Ignore {
				trav, diags := hclsyntax.ParseTraversalAbs([]byte(ignoreStr), "", hcl.Pos{Line: 1, Column: 1})
				if diags.HasErrors() {
					t.Fatalf("failed to parse %q: %s", ignoreStr, diags.Error())
				}
				ignore[i] = trav
			}

			ret, diags := processIgnoreChangesIndividual(test.Old, test.New, ignore)
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}

			if got, want := ret, test.Want; !want.RawEquals(got) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
			}
		})
	}
}

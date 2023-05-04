// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
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
		"list_index": {
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
			[]string{"a[1]"},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.ListVal([]cty.Value{
					cty.StringVal("new a0 value"),
					cty.StringVal("a1 value"),
				}),
				"b": cty.StringVal("new b value"),
			}),
		},
		"map": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("a0 value"),
					"a1": cty.StringVal("a1 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("new a0 value"),
					"a1": cty.UnknownVal(cty.String),
				}),
				"b": cty.StringVal("b value"),
			}),
			[]string{`a`},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("a0 value"),
					"a1": cty.StringVal("a1 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
		},
		"map_index": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("a0 value"),
					"a1": cty.StringVal("a1 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("new a0 value"),
					"a1": cty.StringVal("new a1 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
			[]string{`a["a1"]`},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("new a0 value"),
					"a1": cty.StringVal("a1 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
		},
		"map_index_no_config": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("a0 value"),
					"a1": cty.StringVal("a1 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.Map(cty.String)),
				"b": cty.StringVal("b value"),
			}),
			[]string{`a["a1"]`},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a1": cty.StringVal("a1 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
		},
		"map_index_unknown_value": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("a0 value"),
					"a1": cty.StringVal("a1 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("a0 value"),
					"a1": cty.UnknownVal(cty.String),
				}),
				"b": cty.StringVal("b value"),
			}),
			[]string{`a["a1"]`},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("a0 value"),
					"a1": cty.StringVal("a1 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
		},
		"map_index_multiple_keys": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("a0 value"),
					"a1": cty.StringVal("a1 value"),
					"a2": cty.StringVal("a2 value"),
					"a3": cty.StringVal("a3 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.Map(cty.String)),
				"b": cty.StringVal("new b value"),
			}),
			[]string{`a["a1"]`, `a["a2"]`, `a["a3"]`, `b`},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a1": cty.StringVal("a1 value"),
					"a2": cty.StringVal("a2 value"),
					"a3": cty.StringVal("a3 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
		},
		"map_index_redundant": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("a0 value"),
					"a1": cty.StringVal("a1 value"),
					"a2": cty.StringVal("a2 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.Map(cty.String)),
				"b": cty.StringVal("new b value"),
			}),
			[]string{`a["a1"]`, `a`, `b`},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("a0 value"),
					"a1": cty.StringVal("a1 value"),
					"a2": cty.StringVal("a2 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
		},
		"missing_map_index": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("a0 value"),
					"a1": cty.StringVal("a1 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapValEmpty(cty.String),
				"b": cty.StringVal("b value"),
			}),
			[]string{`a["a1"]`},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a1": cty.StringVal("a1 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
		},
		"missing_map_index_empty": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapValEmpty(cty.String),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a0 value"),
				}),
			}),
			[]string{`a["a"]`},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapValEmpty(cty.String),
			}),
		},
		"missing_map_index_to_object": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"a": cty.StringVal("aa0"),
						"b": cty.StringVal("ab0"),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"a": cty.StringVal("ba0"),
						"b": cty.StringVal("bb0"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapValEmpty(
					cty.Object(map[string]cty.Type{
						"a": cty.String,
						"b": cty.String,
					}),
				),
			}),
			// we expect the config to be used here, as the ignore changes was
			// `a["a"].b`, but the change was larger than that removing
			// `a["a"]` entirely.
			[]string{`a["a"].b`},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapValEmpty(
					cty.Object(map[string]cty.Type{
						"a": cty.String,
						"b": cty.String,
					}),
				),
			}),
		},
		"missing_prior_map_index": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("a0 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("a0 value"),
					"a1": cty.StringVal("new a1 value"),
				}),
				"b": cty.StringVal("b value"),
			}),
			[]string{`a["a1"]`},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.MapVal(map[string]cty.Value{
					"a0": cty.StringVal("a0 value"),
				}),
				"b": cty.StringVal("b value"),
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
		"unknown_object_attribute": {
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
					"bar": cty.UnknownVal(cty.String),
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
		"null_map": {
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("ok"),
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"s":   cty.StringVal("ok"),
						"map": cty.NullVal(cty.Map(cty.String)),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"s":   cty.StringVal("ok"),
						"map": cty.NullVal(cty.Map(cty.String)),
					}),
				}),
			}),
			[]string{"a"},
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("ok"),
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"s":   cty.StringVal("ok"),
						"map": cty.NullVal(cty.Map(cty.String)),
					}),
				}),
			}),
		},
		"marked_map": {
			cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"key": cty.StringVal("val"),
				}).Mark("marked"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"key": cty.StringVal("new val"),
				}).Mark("marked"),
			}),
			[]string{`map["key"]`},
			cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"key": cty.StringVal("val"),
				}).Mark("marked"),
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

			ret, diags := processIgnoreChangesIndividual(test.Old, test.New, traversalsToPaths(ignore))
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}

			if got, want := ret, test.Want; !want.RawEquals(got) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
			}
		})
	}
}

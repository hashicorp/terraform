// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseRemovedFrom(t *testing.T) {

	mustExpr := func(t *testing.T, expr string) hcl.Expression {
		ret, diags := hclsyntax.ParseExpression([]byte(expr), "", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}
		return ret
	}

	tcs := []struct {
		from      string
		component Component
		index     cty.Value
		vars      map[string]cty.Value
		diags     func() tfdiags.Diagnostics
	}{
		{
			from: "component.component_name",
			component: Component{
				Name: "component_name",
			},
		},
		{
			from: "component.component_name[0]",
			component: Component{
				Name: "component_name",
			},
			index: cty.NumberIntVal(0),
		},
		{
			from: "component.component_name[\"key\"]",
			component: Component{
				Name: "component_name",
			},
			index: cty.StringVal("key"),
		},
		{
			from: "component.component_name[each.key]",
			component: Component{
				Name: "component_name",
			},
			index: cty.StringVal("key"),
			vars: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"key": cty.StringVal("key"),
				}),
			},
		},
		{
			from: "component.component_name[each.value.attribute]",
			component: Component{
				Name: "component_name",
			},
			index: cty.StringVal("attribute"),
			vars: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"value": cty.ObjectVal(map[string]cty.Value{
						"attribute": cty.StringVal("attribute"),
					}),
				}),
			},
		},
		{
			from: "component.component_name[each.value[\"key\"]]",
			component: Component{
				Name: "component_name",
			},
			index: cty.StringVal("key"),
			vars: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"value": cty.MapVal(map[string]cty.Value{
						"key": cty.StringVal("key"),
					}),
				}),
			},
		},
		{
			from: "component.component_name[each.value[\"key\"].attribute]",
			component: Component{
				Name: "component_name",
			},
			index: cty.StringVal("attribute"),
			vars: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"value": cty.MapVal(map[string]cty.Value{
						"key": cty.ObjectVal(map[string]cty.Value{
							"attribute": cty.StringVal("attribute"),
						}),
					}),
				}),
			},
		},
		{
			from: "component.component_name[each.value[local.key]]",
			component: Component{
				Name: "component_name",
			},
			index: cty.StringVal("key"),
			vars: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"value": cty.MapVal(map[string]cty.Value{
						"key": cty.StringVal("key"),
					}),
				}),
				"local": cty.ObjectVal(map[string]cty.Value{
					"key": cty.StringVal("key"),
				}),
			},
		},
		{
			from: "component.component_name[each.value[local.key].attribute]",
			component: Component{
				Name: "component_name",
			},
			index: cty.StringVal("attribute"),
			vars: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"value": cty.MapVal(map[string]cty.Value{
						"key": cty.ObjectVal(map[string]cty.Value{
							"attribute": cty.StringVal("attribute"),
						}),
					}),
				}),
				"local": cty.ObjectVal(map[string]cty.Value{
					"key": cty.StringVal("key"),
				}),
			},
		},
		{
			from: "component.component_name.attribute_key",
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'from' attribute",
					Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
				})
				return diags
			},
		},
		{
			from: "component.component_name[0].attribute_key",
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'from' attribute",
					Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
				})
				return diags
			},
		},
		{
			from: "component.component_name[\"key\"].attribute_key",
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'from' attribute",
					Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
				})
				return diags
			},
		},
		{
			from: "component.component_name[each.key].attribute_key",
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'from' attribute",
					Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
				})
				return diags
			},
		},
		{
			from: "component.component_name.attribute_key[0]",
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'from' attribute",
					Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
				})
				return diags
			},
		},
		{
			from: "component[0].component_name",
			diags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'from' attribute",
					Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
				})
				return diags
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.from, func(t *testing.T) {
			expr := mustExpr(t, tc.from)
			component, index, gotDiags := ParseRemovedFrom(expr)

			// validate the component first
			if diff := cmp.Diff(tc.component, component); len(diff) > 0 {
				t.Errorf("unexpected result\n%s", diff)
			}

			// validate the index
			if index == nil {
				if tc.index != cty.NilVal {
					t.Errorf("expected index but got nil")
				}
			} else {
				gotIndex, indexDiags := index.Value(&hcl.EvalContext{
					Variables: tc.vars,
				})
				if len(indexDiags) > 0 {
					t.Errorf("unexpected index diagnostics: %s", indexDiags.Error())
				}
				if diff := cmp.Diff(tc.index, gotIndex, ctydebug.CmpOptions); len(diff) > 0 {
					t.Errorf("unexpected index\n%s", diff)
				}
			}

			// validate the diagnostics

			var wantDiags tfdiags.Diagnostics
			if tc.diags != nil {
				wantDiags = tc.diags()
			}
			if len(gotDiags) != len(wantDiags) {
				t.Errorf("wrong number of diagnostics")
			}
			for ix, got := range gotDiags {
				want := wantDiags[ix]

				if want.Severity() != got.Severity() {
					t.Errorf("unexpected severity: got %s, want %s", got.Severity(), want.Severity())
				}
				if diff := cmp.Diff(want.Description(), got.Description()); len(diff) > 0 {
					t.Errorf("unexpected description\n%s", diff)
				}
			}
		})
	}

}

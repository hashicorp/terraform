// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
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
		from       string
		want       AbsComponentInstance
		vars       map[string]cty.Value
		parseDiags func() tfdiags.Diagnostics
		addrDiags  func() tfdiags.Diagnostics
	}{
		{
			from: "component.component_name",
			want: mustAbsComponentInstance(t, "component.component_name"),
		},
		{
			from: "component.component_name[0]",
			want: mustAbsComponentInstance(t, "component.component_name[0]"),
		},
		{
			from: "component.component_name[\"key\"]",
			want: mustAbsComponentInstance(t, "component.component_name[\"key\"]"),
		},
		{
			from: "component.component_name[each.key]",
			want: mustAbsComponentInstance(t, "component.component_name[\"key\"]"),
			vars: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"key": cty.StringVal("key"),
				}),
			},
		},
		{
			from: "component.component_name[each.value.attribute]",
			want: mustAbsComponentInstance(t, "component.component_name[\"attribute\"]"),
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
			want: mustAbsComponentInstance(t, "component.component_name[\"key\"]"),
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
			want: mustAbsComponentInstance(t, "component.component_name[\"attribute\"]"),
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
			want: mustAbsComponentInstance(t, "component.component_name[\"key\"]"),
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
			want: mustAbsComponentInstance(t, "component.component_name[\"attribute\"]"),
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
			from: "stack.stack_name.component.component_name",
			want: mustAbsComponentInstance(t, "stack.stack_name.component.component_name"),
		},
		{
			from: "stack.parent.stack.child.component.component_name",
			want: mustAbsComponentInstance(t, "stack.parent.stack.child.component.component_name"),
		},
		{
			from: "stack.stack_name[\"stack\"].component.component_name",
			want: mustAbsComponentInstance(t, "stack.stack_name[\"stack\"].component.component_name"),
		},
		{
			from: "stack.stack_name.component.component_name[\"component\"]",
			want: mustAbsComponentInstance(t, "stack.stack_name.component.component_name[\"component\"]"),
		},
		{
			from: "stack.stack_name[\"stack\"].component.component_name[\"component\"]",
			want: mustAbsComponentInstance(t, "stack.stack_name[\"stack\"].component.component_name[\"component\"]"),
		},
		{
			from: "stack.stack_name.component.component_name[each.value]",
			want: mustAbsComponentInstance(t, "stack.stack_name.component.component_name[\"component\"]"),
			vars: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("component"),
				}),
			},
		},
		{
			from: "stack.stack_name[\"stack\"].component.component_name[each.value]",
			want: mustAbsComponentInstance(t, "stack.stack_name[\"stack\"].component.component_name[\"component\"]"),
			vars: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("component"),
				}),
			},
		},
		{
			from: "stack.stack_name[each.value].component.component_name",
			want: mustAbsComponentInstance(t, "stack.stack_name[\"stack\"].component.component_name"),
			vars: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("stack"),
				}),
			},
		},
		{
			from: "stack.stack_name[each.value].component.component_name[\"component\"]",
			want: mustAbsComponentInstance(t, "stack.stack_name[\"stack\"].component.component_name[\"component\"]"),
			vars: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("stack"),
				}),
			},
		},
		{
			from: "stack.stack_name[each.value[\"stack\"]].component.component_name[each.value[\"component\"]]",
			want: mustAbsComponentInstance(t, "stack.stack_name[\"stack\"].component.component_name[\"component\"]"),
			vars: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"value": cty.ObjectVal(map[string]cty.Value{
						"stack":     cty.StringVal("stack"),
						"component": cty.StringVal("component"),
					}),
				}),
			},
		},
		{
			from: "stack.parent[each.value[\"parent\"]].stack.child[each.value[\"child\"]].component.component_name",
			want: mustAbsComponentInstance(t, "stack.parent[\"parent\"].stack.child[\"child\"].component.component_name"),
			vars: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"value": cty.ObjectVal(map[string]cty.Value{
						"parent":    cty.StringVal("parent"),
						"child":     cty.StringVal("child"),
						"component": cty.StringVal("component"),
					}),
				}),
			},
		},
		{
			from: "stack.parent[each.value[\"parent\"]].stack.child[each.value[\"child\"]].component.component_name[\"component\"]",
			want: mustAbsComponentInstance(t, "stack.parent[\"parent\"].stack.child[\"child\"].component.component_name[\"component\"]"),
			vars: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"value": cty.ObjectVal(map[string]cty.Value{
						"parent": cty.StringVal("parent"),
						"child":  cty.StringVal("child"),
					}),
				}),
			},
		},
		{
			from: "stack.parent[each.value[\"parent\"]].stack.child[each.value[\"child\"]].component.component_name[each.value[\"component\"]]",
			want: mustAbsComponentInstance(t, "stack.parent[\"parent\"].stack.child[\"child\"].component.component_name[\"component\"]"),
			vars: map[string]cty.Value{
				"each": cty.ObjectVal(map[string]cty.Value{
					"value": cty.ObjectVal(map[string]cty.Value{
						"parent":    cty.StringVal("parent"),
						"child":     cty.StringVal("child"),
						"component": cty.StringVal("component"),
					}),
				}),
			},
		},
		{
			from: "component.component_name.attribute_key",
			parseDiags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'from' attribute",
					Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
					Subject: &hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:   hcl.Pos{Line: 1, Column: 39, Byte: 38},
					},
				})
				return diags
			},
		},
		{
			from: "component.component_name[0].attribute_key",
			parseDiags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'from' attribute",
					Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
					Subject: &hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:   hcl.Pos{Line: 1, Column: 42, Byte: 41},
					},
				})
				return diags
			},
		},
		{
			from: "component.component_name[\"key\"].attribute_key",
			parseDiags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'from' attribute",
					Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
					Subject: &hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:   hcl.Pos{Line: 1, Column: 46, Byte: 45},
					},
				})
				return diags
			},
		},
		{
			from: "component.component_name[each.key].attribute_key",
			parseDiags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'from' attribute",
					Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
					Subject: &hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:   hcl.Pos{Line: 1, Column: 49, Byte: 48},
					},
				})
				return diags
			},
		},
		{
			from: "component.component_name.attribute_key[0]",
			parseDiags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'from' attribute",
					Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
					Subject: &hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:   hcl.Pos{Line: 1, Column: 42, Byte: 41},
					},
				})
				return diags
			},
		},
		{
			from: "component[0].component_name",
			parseDiags: func() tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid 'from' attribute",
					Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
					Subject: &hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:   hcl.Pos{Line: 1, Column: 28, Byte: 27},
					},
				})
				return diags
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.from, func(t *testing.T) {
			expr := mustExpr(t, tc.from)
			from, parseDiags := ParseRemovedFrom(expr)

			var wantParseDiags tfdiags.Diagnostics
			if tc.parseDiags != nil {
				wantParseDiags = tc.parseDiags()
			}
			tfdiags.AssertDiagnosticsMatch(t, parseDiags, wantParseDiags)

			configAddress := from.ConfigComponent()
			instanceAddress, addrDiags := from.AbsComponentInstance(&hcl.EvalContext{
				Variables: tc.vars,
			}, RootStackInstance)
			var wantAddrDiags tfdiags.Diagnostics
			if tc.addrDiags != nil {
				wantAddrDiags = tc.addrDiags()
			}
			tfdiags.AssertDiagnosticsMatch(t, addrDiags, wantAddrDiags)

			wantConfigAddress := ConfigComponent{
				Stack: tc.want.Stack.ConfigAddr(),
				Item:  tc.want.Item.Component,
			}
			if diff := cmp.Diff(configAddress.String(), wantConfigAddress.String()); len(diff) > 0 {
				t.Errorf("wrong config address; %s", diff)
			}
			if diff := cmp.Diff(instanceAddress.String(), tc.want.String()); len(diff) > 0 {
				t.Errorf("wrong instance address: %s", diff)
			}
		})
	}

}

func mustAbsComponentInstance(t *testing.T, str string) AbsComponentInstance {
	inst, diags := ParseAbsComponentInstanceStr(str)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}
	return inst
}

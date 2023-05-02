// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/zclconf/go-cty/cty"
)

func TestEvalReplaceTriggeredBy(t *testing.T) {
	tests := map[string]struct {
		// Raw config expression from within replace_triggered_by list.
		// If this does not contains any count or each references, it should
		// directly parse into the same *addrs.Reference.
		expr string

		// If the expression contains count or each, then we need to add
		// repetition data, and the static string to parse into the desired
		// *addrs.Reference
		repData   instances.RepetitionData
		reference string
	}{
		"single resource": {
			expr: "test_resource.a",
		},

		"resource instance attr": {
			expr: "test_resource.a.attr",
		},

		"resource instance index attr": {
			expr: "test_resource.a[0].attr",
		},

		"resource instance count": {
			expr: "test_resource.a[count.index]",
			repData: instances.RepetitionData{
				CountIndex: cty.NumberIntVal(0),
			},
			reference: "test_resource.a[0]",
		},
		"resource instance for_each": {
			expr: "test_resource.a[each.key].attr",
			repData: instances.RepetitionData{
				EachKey: cty.StringVal("k"),
			},
			reference: `test_resource.a["k"].attr`,
		},
		"resource instance for_each map attr": {
			expr: "test_resource.a[each.key].attr[each.key]",
			repData: instances.RepetitionData{
				EachKey: cty.StringVal("k"),
			},
			reference: `test_resource.a["k"].attr["k"]`,
		},
	}

	for name, tc := range tests {
		pos := hcl.Pos{Line: 1, Column: 1}
		t.Run(name, func(t *testing.T) {
			expr, hclDiags := hclsyntax.ParseExpression([]byte(tc.expr), "", pos)
			if hclDiags.HasErrors() {
				t.Fatal(hclDiags)
			}

			got, diags := evalReplaceTriggeredByExpr(expr, tc.repData)
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}

			want := tc.reference
			if want == "" {
				want = tc.expr
			}

			// create the desired reference
			traversal, travDiags := hclsyntax.ParseTraversalAbs([]byte(want), "", pos)
			if travDiags.HasErrors() {
				t.Fatal(travDiags)
			}
			ref, diags := addrs.ParseRef(traversal)
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}

			if got.DisplayString() != ref.DisplayString() {
				t.Fatalf("expected %q: got %q", ref.DisplayString(), got.DisplayString())
			}
		})
	}
}

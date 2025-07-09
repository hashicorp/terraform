package configs

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
)

func TestDecodeActionBlock(t *testing.T) {
	fooResourceExpr := hcltest.MockExprTraversalSrc("resource_type.foo")
	barResourceExpr := hcltest.MockExprTraversalSrc("resource_type.bar")

	fooAndBarExpr := hcltest.MockExprList([]hcl.Expression{fooResourceExpr, barResourceExpr})

	tests := map[string]struct {
		input       *hcl.Block
		want        *Action
		expectDiags []hcl.Diagnostic
	}{
		"one linked resource": {
			&hcl.Block{
				Type:   "action",
				Labels: []string{"an_action", "foo"},
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"linked_resource": {
							Name: "linked_resource",
							Expr: fooResourceExpr,
						},
					},
				}),
				DefRange:    blockRange,
				LabelRanges: []hcl.Range{{}},
			},
			&Action{
				Type:            "an_action",
				Name:            "foo",
				LinkedResources: []hcl.Traversal{mustAbsTraversalForExpr(fooResourceExpr)},
				DeclRange:       blockRange,
			},
			nil,
		},
		"multiple linked resources": {
			&hcl.Block{
				Type:   "action",
				Labels: []string{"an_action", "foo"},
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"linked_resources": {
							Name: "linked_resources",
							Expr: fooAndBarExpr,
						},
					},
				}),
				DefRange:    blockRange,
				LabelRanges: []hcl.Range{{}},
			},
			&Action{
				Type:            "an_action",
				Name:            "foo",
				LinkedResources: []hcl.Traversal{mustAbsTraversalForExpr(fooResourceExpr), mustAbsTraversalForExpr(barResourceExpr)},
				DeclRange:       blockRange,
			},
			nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, diags := decodeActionBlock(test.input)
			if len(diags) != len(test.expectDiags) {
				t.Error(diags.Error())
				t.Fatalf("Wrong result! Expected %d diagnostics, got %d", len(test.expectDiags), len(diags))
			}

			assertResultDeepEqual(t, got, test.want)
		})
	}
}

func mustAbsTraversalForExpr(expr hcl.Expression) hcl.Traversal {
	trav, diags := hcl.AbsTraversalForExpr(expr)
	if diags.HasErrors() {
		panic(diags.Errs())
	}
	return trav
}

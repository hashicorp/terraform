// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

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
	moduleResourceExpr := hcltest.MockExprTraversalSrc("module.foo.resource_type.bar")
	fooDataSourceExpr := hcltest.MockExprTraversalSrc("data.example.foo")

	tests := map[string]struct {
		input       *hcl.Block
		want        *Action
		expectDiags []string
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
		"invalid linked resources (module ref)": { // for now! This test will change when we support cross-module actions
			&hcl.Block{
				Type:   "action",
				Labels: []string{"an_action", "foo"},
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"linked_resources": {
							Name: "linked_resources",
							Expr: hcltest.MockExprList([]hcl.Expression{moduleResourceExpr}),
						},
					},
				}),
				DefRange:    blockRange,
				LabelRanges: []hcl.Range{{}},
			},
			&Action{
				Type:            "an_action",
				Name:            "foo",
				LinkedResources: []hcl.Traversal{},
				DeclRange:       blockRange,
			},
			[]string{`:0,0-0: Invalid "linked_resources"; "linked_resources" must only refer to managed resources in the current module.`},
		},
		"invalid linked resource (datasource ref)": {
			&hcl.Block{
				Type:   "action",
				Labels: []string{"an_action", "foo"},
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"linked_resource": {
							Name: "linked_resource",
							Expr: fooDataSourceExpr,
						},
					},
				}),
				DefRange:    blockRange,
				LabelRanges: []hcl.Range{{}},
			},
			&Action{
				Type:            "an_action",
				Name:            "foo",
				LinkedResources: nil,
				DeclRange:       blockRange,
			},
			[]string{`:0,0-0: Invalid "linked_resource"; "linked_resource" must only refer to a managed resource in the current module.`},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, diags := decodeActionBlock(test.input)
			assertExactDiagnostics(t, diags, test.expectDiags)
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

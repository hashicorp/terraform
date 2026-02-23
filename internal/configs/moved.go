// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
)

type Moved struct {
	From *addrs.MoveEndpoint
	To   *addrs.MoveEndpoint

	// FromExpr and ToExpr retain the original endpoint expressions so moved
	// blocks with for_each can be expanded later using each.key/each.value.
	FromExpr hcl.Expression
	ToExpr   hcl.Expression

	// ForEach is reserved for future moved-block expansion support.
	// Terraform doesn't evaluate it yet, but we retain the expression so the
	// moved mini-graph can eventually analyze references and evaluate it.
	ForEach hcl.Expression

	DeclRange hcl.Range
}

func decodeMovedBlock(block *hcl.Block) (*Moved, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	moved := &Moved{
		DeclRange: block.DefRange,
	}

	content, moreDiags := block.Body.Content(movedBlockSchema)
	diags = append(diags, moreDiags...)

	if attr, exists := content.Attributes["for_each"]; exists {
		moved.ForEach = attr.Expr
	}

	if attr, exists := content.Attributes["from"]; exists {
		moved.FromExpr = attr.Expr
		from, fromDiags := parseMoveEndpointExpr(attr.Expr, moved.ForEach != nil)
		diags = append(diags, fromDiags...)
		moved.From = from
	}

	if attr, exists := content.Attributes["to"]; exists {
		moved.ToExpr = attr.Expr
		to, toDiags := parseMoveEndpointExpr(attr.Expr, moved.ForEach != nil)
		diags = append(diags, toDiags...)
		moved.To = to
	}

	// we can only move from a module to a module, resource to resource, etc.
	if !diags.HasErrors() {
		if !moved.From.MightUnifyWith(moved.To) {
			// We can catch some obviously-wrong combinations early here,
			// but we still have other dynamic validation to do at runtime.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid \"moved\" addresses",
				Detail:   "The \"from\" and \"to\" addresses must either both refer to resources or both refer to modules.",
				Subject:  &moved.DeclRange,
			})
		}
	}

	return moved, diags
}

func parseMoveEndpointExpr(expr hcl.Expression, allowDynamicIndexes bool) (*addrs.MoveEndpoint, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	var traversal hcl.Traversal
	if allowDynamicIndexes {
		var traversalDiags hcl.Diagnostics
		traversal, traversalDiags = exprToTraversalSkippingIndexes(expr)
		diags = append(diags, traversalDiags...)
		if traversalDiags.HasErrors() {
			return nil, diags
		}
	} else {
		var traversalDiags hcl.Diagnostics
		traversal, traversalDiags = hcl.AbsTraversalForExpr(expr)
		diags = append(diags, traversalDiags...)
		if traversalDiags.HasErrors() {
			return nil, diags
		}
	}

	ep, epDiags := addrs.ParseMoveEndpoint(traversal)
	diags = append(diags, epDiags.ToHCL()...)
	if epDiags.HasErrors() {
		return nil, diags
	}
	return ep, diags
}

var movedBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name:     "from",
			Required: true,
		},
		{
			Name:     "to",
			Required: true,
		},
		{
			Name: "for_each",
		},
	},
}

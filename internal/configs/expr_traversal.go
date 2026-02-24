// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// exprToTraversalSkippingIndexes parses a traversal-like expression while
// omitting index expressions. Callers can then interpret the returned traversal
// as a "shape" address and evaluate index expressions later.
func exprToTraversalSkippingIndexes(expr hcl.Expression) (hcl.Traversal, hcl.Diagnostics) {
	var trav hcl.Traversal
	var diags hcl.Diagnostics

	switch e := expr.(type) {
	case *hclsyntax.RelativeTraversalExpr:
		t, d := exprToTraversalSkippingIndexes(e.Source)
		diags = append(diags, d...)
		if d.HasErrors() {
			return nil, diags
		}
		trav = append(trav, t...)
		trav = append(trav, e.Traversal...)

	case *hclsyntax.ScopeTraversalExpr:
		trav = append(trav, e.Traversal...)

	case *hclsyntax.IndexExpr:
		t, d := exprToTraversalSkippingIndexes(e.Collection)
		diags = append(diags, d...)
		if d.HasErrors() {
			return nil, diags
		}
		trav = append(trav, t...)

	default:
		t, d := hcl.AbsTraversalForExpr(e)
		diags = append(diags, d...)
		if d.HasErrors() {
			return nil, diags
		}
		trav = append(trav, t...)
	}

	return trav, diags
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func evalReplaceTriggeredByExpr(expr hcl.Expression, keyData instances.RepetitionData) (*addrs.Reference, tfdiags.Diagnostics) {
	var ref *addrs.Reference
	var diags tfdiags.Diagnostics

	traversal, diags := triggersExprToTraversal(expr, keyData)
	if diags.HasErrors() {
		return nil, diags
	}

	// We now have a static traversal, so we can just turn it into an addrs.Reference.
	ref, ds := addrs.ParseRef(traversal)
	diags = diags.Append(ds)

	return ref, diags
}

// trggersExprToTraversal takes an hcl expression limited to the syntax allowed
// in replace_triggered_by, and converts it to a static traversal. The
// RepetitionData contains the data necessary to evaluate the only allowed
// variables in the expression, count.index and each.key.
func triggersExprToTraversal(expr hcl.Expression, keyData instances.RepetitionData) (hcl.Traversal, tfdiags.Diagnostics) {
	var trav hcl.Traversal
	var diags tfdiags.Diagnostics

	switch e := expr.(type) {
	case *hclsyntax.RelativeTraversalExpr:
		t, d := triggersExprToTraversal(e.Source, keyData)
		diags = diags.Append(d)
		trav = append(trav, t...)
		trav = append(trav, e.Traversal...)

	case *hclsyntax.ScopeTraversalExpr:
		// a static reference, we can just append the traversal
		trav = append(trav, e.Traversal...)

	case *hclsyntax.IndexExpr:
		// Get the collection from the index expression
		t, d := triggersExprToTraversal(e.Collection, keyData)
		diags = diags.Append(d)
		if diags.HasErrors() {
			return nil, diags
		}
		trav = append(trav, t...)

		// The index key is the only place where we could have variables that
		// reference count and each, so we need to parse those independently.
		idx, hclDiags := parseIndexKeyExpr(e.Key, keyData)
		diags = diags.Append(hclDiags)

		trav = append(trav, idx)

	default:
		// Something unexpected got through config validation. We're not sure
		// what it is, but we'll point it out in the diagnostics for the user
		// to fix.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid replace_triggered_by expression",
			Detail:   "Unexpected expression found in replace_triggered_by.",
			Subject:  e.Range().Ptr(),
		})
	}

	return trav, diags
}

// parseIndexKeyExpr takes an hcl.Expression and parses it as an index key, while
// evaluating any references to count.index or each.key.
func parseIndexKeyExpr(expr hcl.Expression, keyData instances.RepetitionData) (hcl.TraverseIndex, hcl.Diagnostics) {
	idx := hcl.TraverseIndex{
		SrcRange: expr.Range(),
	}

	trav, diags := hcl.RelTraversalForExpr(expr)
	if diags.HasErrors() {
		return idx, diags
	}

	keyParts := []string{}

	for _, t := range trav {
		attr, ok := t.(hcl.TraverseAttr)
		if !ok {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid index expression",
				Detail:   "Only constant values, count.index or each.key are allowed in index expressions.",
				Subject:  expr.Range().Ptr(),
			})
			return idx, diags
		}
		keyParts = append(keyParts, attr.Name)
	}

	switch strings.Join(keyParts, ".") {
	case "count.index":
		if keyData.CountIndex == cty.NilVal {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Reference to "count" in non-counted context`,
				Detail:   `The "count" object can only be used in "resource" blocks when the "count" argument is set.`,
				Subject:  expr.Range().Ptr(),
			})
		}
		idx.Key = keyData.CountIndex

	case "each.key":
		if keyData.EachKey == cty.NilVal {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Reference to "each" in context without for_each`,
				Detail:   `The "each" object can be used only in "resource" blocks when the "for_each" argument is set.`,
				Subject:  expr.Range().Ptr(),
			})
		}
		idx.Key = keyData.EachKey
	default:
		// Something may have slipped through validation, probably from a json
		// configuration.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid index expression",
			Detail:   "Only constant values, count.index or each.key are allowed in index expressions.",
			Subject:  expr.Range().Ptr(),
		})
	}

	return idx, diags

}

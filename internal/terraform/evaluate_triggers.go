// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// evalSemiStaticExpr takes a reference expression which might contain instance
// indexes derived from variable values, and returns an addrs.Reference for the
// known parts.
func evalSemiStaticExpr(expr hcl.Expression, keyData instances.RepetitionData) (*addrs.Reference, tfdiags.Diagnostics) {
	var ref *addrs.Reference
	var diags tfdiags.Diagnostics

	traversal, diags := semiStaticExpToTraversal(expr, keyData)
	if diags.HasErrors() {
		return nil, diags
	}

	// We now have a static traversal, so we can just turn it into an addrs.Reference.
	ref, ds := addrs.ParseRef(traversal)
	diags = diags.Append(ds)

	return ref, diags
}

// semiStaticExpToTraversal takes an hcl expression limited to the syntax allowed
// for static references with dynamic instance keys, and converts it to a static
// traversal. The RepetitionData contains the data necessary to evaluate the
// only allowed variables in the expression, count.index and each.key.
func semiStaticExpToTraversal(expr hcl.Expression, keyData instances.RepetitionData) (hcl.Traversal, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	addrExpr, hclDiags := addrs.ParseAddressExpr(expr)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return nil, diags
	}

	traversal, hclDiags := addrExpr.Traversal(func(key hcl.Expression) (cty.Value, hcl.Diagnostics) {
		return evalSemiStaticKeyExpr(key, keyData)
	})
	diags = diags.Append(hclDiags)
	return traversal, diags
}

// evalSemiStaticKeyExpr takes an hcl.Expression and evaluates it as an index
// key, where the only allowed references are count.index and each.key.
func evalSemiStaticKeyExpr(expr hcl.Expression, keyData instances.RepetitionData) (cty.Value, hcl.Diagnostics) {
	trav, diags := hcl.RelTraversalForExpr(expr)
	if diags.HasErrors() {
		return cty.NilVal, diags
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
			return cty.NilVal, diags
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
		return keyData.CountIndex, diags

	case "each.key":
		if keyData.EachKey == cty.NilVal {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Reference to "each" in context without for_each`,
				Detail:   `The "each" object can be used only in "resource" blocks when the "for_each" argument is set.`,
				Subject:  expr.Range().Ptr(),
			})
		}
		return keyData.EachKey, diags
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

	return cty.NilVal, diags
}

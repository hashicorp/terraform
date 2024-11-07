// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func evaluateImportIdExpression(expr hcl.Expression, ctx EvalContext, keyData instances.RepetitionData, allowUnknown bool) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if expr == nil {
		return cty.NilVal, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid import id argument",
			Detail:   "The import ID cannot be null.",
			Subject:  expr.Range().Ptr(),
		})
	}

	// import blocks only exist in the root module, and must be evaluated in
	// that context.
	ctx = evalContextForModuleInstance(ctx, addrs.RootModuleInstance)
	scope := ctx.EvaluationScope(nil, nil, keyData)
	importIdVal, evalDiags := scope.EvalExpr(expr, cty.String)
	diags = diags.Append(evalDiags)

	if importIdVal.IsNull() {
		return cty.NilVal, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid import id argument",
			Detail:   "The import ID cannot be null.",
			Subject:  expr.Range().Ptr(),
		})
	}
	if !allowUnknown && !importIdVal.IsKnown() {
		return cty.NilVal, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid import id argument",
			Detail:   `The import block "id" argument depends on resource attributes that cannot be determined until apply, so Terraform cannot plan to import this resource.`, // FIXME and what should I do about that?
			Subject:  expr.Range().Ptr(),
			//	Expression:
			//	EvalContext:
			Extra: diagnosticCausedByUnknown(true),
		})
	}

	// Import data may have marks, which we can discard because the id is only
	// sent to the provider.
	importIdVal, _ = importIdVal.Unmark()

	if importIdVal.Type() != cty.String {
		return cty.NilVal, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid import id argument",
			Detail:   "The import ID value is unsuitable: not a string.",
			Subject:  expr.Range().Ptr(),
		})
	}

	if importIdVal.IsKnown() && importIdVal.AsString() == "" {
		return cty.NilVal, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid import id argument",
			Detail:   "The import ID value evaluates to an empty string, please provide a non-empty value.",
			Subject:  expr.Range().Ptr(),
		})
	}

	return importIdVal, diags
}

func evalImportToExpression(expr hcl.Expression, keyData instances.RepetitionData) (addrs.AbsResourceInstance, tfdiags.Diagnostics) {
	var res addrs.AbsResourceInstance
	var diags tfdiags.Diagnostics

	traversal, diags := importToExprToTraversal(expr, keyData)
	if diags.HasErrors() {
		return res, diags
	}

	target, targetDiags := addrs.ParseTarget(traversal)
	diags = diags.Append(targetDiags)
	if diags.HasErrors() {
		return res, targetDiags
	}

	switch sub := target.Subject.(type) {
	case addrs.AbsResource:
		res = sub.Instance(addrs.NoKey)
	case addrs.AbsResourceInstance:
		res = sub
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid import 'to' expression",
			Detail:   fmt.Sprintf("The import block 'to' argument %s does not resolve to a single resource instance.", sub),
			Subject:  expr.Range().Ptr(),
		})
	}

	return res, diags
}

func evalImportUnknownToExpression(expr hcl.Expression) (addrs.PartialExpandedResource, tfdiags.Diagnostics) {
	var per addrs.PartialExpandedResource
	var diags tfdiags.Diagnostics

	traversal, diags := importToExprToTraversal(expr, instances.UnknownForEachRepetitionData(cty.DynamicPseudoType))
	if diags.HasErrors() {
		return per, diags
	}

	per, moreDiags := parseImportToPartialAddress(traversal)
	diags = diags.Append(moreDiags)
	return per, diags
}

// trggersExprToTraversal takes an hcl expression limited to the syntax allowed
// in replace_triggered_by, and converts it to a static traversal. The
// RepetitionData contains the data necessary to evaluate the only allowed
// variables in the expression, count.index and each.key.
func importToExprToTraversal(expr hcl.Expression, keyData instances.RepetitionData) (hcl.Traversal, tfdiags.Diagnostics) {
	var trav hcl.Traversal
	var diags tfdiags.Diagnostics

	switch e := expr.(type) {
	case *hclsyntax.RelativeTraversalExpr:
		t, d := importToExprToTraversal(e.Source, keyData)
		diags = diags.Append(d)
		trav = append(trav, t...)
		trav = append(trav, e.Traversal...)

	case *hclsyntax.ScopeTraversalExpr:
		// a static reference, we can just append the traversal
		trav = append(trav, e.Traversal...)

	case *hclsyntax.IndexExpr:
		// Get the collection from the index expression
		t, d := importToExprToTraversal(e.Collection, keyData)
		diags = diags.Append(d)
		if diags.HasErrors() {
			return nil, diags
		}
		trav = append(trav, t...)

		// The index key is the only place where we could have variables that
		// reference count and each, so we need to parse those independently.
		idx, hclDiags := parseImportToKeyExpression(e.Key, keyData)
		diags = diags.Append(hclDiags)

		trav = append(trav, idx)

	default:
		// if we don't recognise the expression type (which means we are likely
		// dealing with a test mock), try and interpret this as an absolute
		// traversal
		t, d := hcl.AbsTraversalForExpr(e)
		diags = diags.Append(d)
		trav = append(trav, t...)
	}

	return trav, diags
}

// parseImportToKeyExpression takes an hcl.Expression and parses it as an index key, while
// evaluating any references to count.index or each.key.
func parseImportToKeyExpression(expr hcl.Expression, keyData instances.RepetitionData) (hcl.TraverseIndex, hcl.Diagnostics) {
	idx := hcl.TraverseIndex{
		SrcRange: expr.Range(),
	}

	ctx := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"each": cty.ObjectVal(map[string]cty.Value{
				"key":   keyData.EachKey,
				"value": keyData.EachValue,
			}),
		},
	}

	val, diags := expr.Value(ctx)
	if diags.HasErrors() {
		// catch the most common case of an unsupported variable and try to
		// give the user a slightly more helpful error
		for i := range diags {
			if diags[i].Summary == "Unknown variable" {
				diags[i].Detail += "Only \"each.key\" and \"each.value\" can be used in import address index expressions."
			}
		}

		return idx, diags
	}

	if val.HasMark(marks.Sensitive) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid index expression",
			Detail:   "Import address index expression cannot be sensitive.",
			Subject:  expr.Range().Ptr(),
		})
		return idx, diags
	}

	idx.Key = val
	return idx, nil

}

func parseImportToPartialAddress(traversal hcl.Traversal) (addrs.PartialExpandedResource, tfdiags.Diagnostics) {
	partial, rest, diags := addrs.ParsePartialExpandedResource(traversal)
	if diags.HasErrors() {
		return addrs.PartialExpandedResource{}, diags
	}

	if len(rest) > 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid import 'to' expression",
			Detail:   "The import block 'to' argument does not resolve to a single resource instance.",
			Subject:  traversal.SourceRange().Ptr(),
		})
	}

	return partial, diags
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// evaluateCountExpression is our standard mechanism for interpreting an
// expression given for a "count" argument on a resource or a module. This
// should be called during expansion in order to determine the final count
// value.
//
// evaluateCountExpression differs from evaluateCountExpressionValue by
// returning an error if the count value is not known, and converting the
// cty.Value to an integer.
//
// If allowUnknown is false then this function will return error diagnostics
// whenever the expression returns an unknown value. Setting allowUnknown to
// true instead permits unknown values, indicating them by returning the
// placeholder value -1. Callers can assume that a return value of -1 without
// any error diagnostics represents a valid unknown value.
func evaluateCountExpression(expr hcl.Expression, ctx EvalContext, allowUnknown bool) (int, tfdiags.Diagnostics) {
	countVal, diags := evaluateCountExpressionValue(expr, ctx)
	if !allowUnknown && !countVal.IsKnown() {
		// Currently this is a rather bad outcome from a UX standpoint, since we have
		// no real mechanism to deal with this situation and all we can do is produce
		// an error message.
		// FIXME: In future, implement a built-in mechanism for deferring changes that
		// can't yet be predicted, and use it to guide the user through several
		// plan/apply steps until the desired configuration is eventually reached.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid count argument",
			Detail:   `The "count" value depends on resource attributes that cannot be determined until apply, so Terraform cannot predict how many instances will be created. To work around this, use the -target argument to first apply only the resources that the count depends on.`,
			Subject:  expr.Range().Ptr(),

			// TODO: Also populate Expression and EvalContext in here, but
			// we can't easily do that right now because the hcl.EvalContext
			// (which is not the same as the ctx we have in scope here) is
			// hidden away inside evaluateCountExpressionValue.
			Extra: diagnosticCausedByUnknown(true),
		})
	}

	// Ephemeral values are not allowed in count expressions.
	if countVal.HasMark(marks.Ephemeral) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid count argument",
			Detail:   `The given "count" value is derived from an ephemeral value, which means that Terraform cannot persist it between plan/apply rounds. Use only non-ephemeral values here.`,
			Subject:  expr.Range().Ptr(),
			Extra:    DiagnosticCausedByEphemeral(true),
		})
		return -1, diags
	}

	if countVal.IsNull() || !countVal.IsKnown() {
		return -1, diags
	}

	count, _ := countVal.AsBigFloat().Int64()
	return int(count), diags
}

// evaluateCountExpressionValue is like evaluateCountExpression
// except that it returns a cty.Value which must be a cty.Number and can be
// unknown.
func evaluateCountExpressionValue(expr hcl.Expression, ctx EvalContext) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	nullCount := cty.NullVal(cty.Number)
	if expr == nil {
		return nullCount, nil
	}

	countVal, countDiags := ctx.EvaluateExpr(expr, cty.Number, nil)
	diags = diags.Append(countDiags)
	if diags.HasErrors() {
		return nullCount, diags
	}

	if countVal.HasMark(marks.Ephemeral) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid count argument",
			Detail:   `The given "count" is derived from an ephemeral value, which means that Terraform cannot persist it between plan/apply rounds. Use only non-ephemeral values to specify the number of resource instances.`,
			Subject:  expr.Range().Ptr(),

			// TODO: Also populate Expression and EvalContext in here, but
			// we can't easily do that right now because the hcl.EvalContext
			// (which is not the same as the ctx we have in scope here) is
			// hidden away inside ctx.EvaluateExpr.
			Extra: DiagnosticCausedByEphemeral(true),
		})
	}

	// Sensitive values are allowed in count but not for_each. This is a
	// somewhat-dubious decision because the number of instances planned
	// will disclose exactly what the value was, but in practice it's rare
	// for a number alone to be sensitive and so this is pragmatic, along with
	// being required for backward-compatibility.
	countVal, _ = countVal.Unmark()

	switch {
	case countVal.IsNull():
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid count argument",
			Detail:   `The given "count" argument value is null. An integer is required.`,
			Subject:  expr.Range().Ptr(),
		})
		return nullCount, diags

	case !countVal.IsKnown():
		return cty.UnknownVal(cty.Number), diags
	}

	var count int
	err := gocty.FromCtyValue(countVal, &count)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid count argument",
			Detail:   fmt.Sprintf(`The given "count" argument value is unsuitable: %s.`, err),
			Subject:  expr.Range().Ptr(),
		})
		return nullCount, diags
	}
	if count < 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid count argument",
			Detail:   `The given "count" argument value is unsuitable: must be greater than or equal to zero.`,
			Subject:  expr.Range().Ptr(),
		})
		return nullCount, diags
	}

	return countVal, diags
}

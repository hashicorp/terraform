// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// evaluateCountExpression is our standard mechanism for interpreting an
// expression given for a "count" argument on a resource or a module. This
// should be called during expansion in order to determine the final count
// value.
//
// evaluateCountExpression differs from evaluateCountExpressionValue by
// returning an error if the count value is not known, and converting the
// cty.Value to an integer.
func evaluateCountExpression(expr hcl.Expression, ctx EvalContext) (v int, known bool, diags tfdiags.Diagnostics) {
	countVal, diags := evaluateCountExpressionValue(expr, ctx)

	if !countVal.IsKnown() {
		return -1, false, diags
	}
	if countVal.IsNull() {
		return -1, true, diags
	}

	count, _ := countVal.AsBigFloat().Int64()
	return int(count), true, diags
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

	// Unmark the count value, sensitive values are allowed in count but not for_each,
	// as using it here will not disclose the sensitive value
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

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

// evaluateLimitExpression will evaluate the limit expression within a list block.
//
// If the expression is nil, it will return a default limit of 100.
func evaluateLimitExpression(expr hcl.Expression, ctx EvalContext) (int64, tfdiags.Diagnostics) {
	defaultLimit := int64(100)
	var diags tfdiags.Diagnostics

	if expr == nil {
		return defaultLimit, diags
	}

	limitVal, limitDiags := ctx.EvaluateExpr(expr, cty.Number, nil)
	diags = diags.Append(limitDiags)
	if diags.HasErrors() {
		return defaultLimit, diags
	}

	switch {
	case limitVal.IsNull():
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid limit argument",
			Detail:   `The given "limit" argument value is null. An integer is required.`,
			Subject:  expr.Range().Ptr(),
		})
		return defaultLimit, diags
	case !limitVal.IsKnown():
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid limit argument",
			Detail:   `The given "limit" argument value is unknown. A known integer is required.`,
			Subject:  expr.Range().Ptr(),
		})
		return defaultLimit, diags
	}

	var limit int64
	err := gocty.FromCtyValue(limitVal, &limit)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid limit argument",
			Detail:   fmt.Sprintf(`The given "limit" argument value is unsuitable: %s.`, err),
			Subject:  expr.Range().Ptr(),
		})
		return defaultLimit, diags
	}
	if limit < 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid limit argument",
			Detail:   `The given "limit" argument value is unsuitable: must be greater than or equal to zero.`,
			Subject:  expr.Range().Ptr(),
		})
		return defaultLimit, diags
	}

	return limit, diags
}

// evaluateIncludeResourceExpression will evaluate the include_resource expression within a list block.
//
// If the expression is nil, it will return false, indicating that resources should not be included by default.
func evaluateIncludeResourceExpression(expr hcl.Expression, ctx EvalContext) (bool, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if expr == nil {
		return false, diags
	}

	includeVal, includeDiags := ctx.EvaluateExpr(expr, cty.Bool, nil)
	diags = diags.Append(includeDiags)
	if diags.HasErrors() {
		return false, diags
	}

	switch {
	case includeVal.IsNull():
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid include_resource argument",
			Detail:   `The given "include_resource" argument value is null. A boolean is required.`,
			Subject:  expr.Range().Ptr(),
		})
		return false, diags
	case !includeVal.IsKnown():
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid include_resource argument",
			Detail:   `The given "include_resource" argument value is unknown. A known boolean is required.`,
			Subject:  expr.Range().Ptr(),
		})
		return false, diags
	}

	var include bool
	err := gocty.FromCtyValue(includeVal, &include)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid include_resource argument",
			Detail:   fmt.Sprintf(`The given "include_resource" argument value is unsuitable: %s.`, err),
			Subject:  expr.Range().Ptr(),
		})
		return false, diags
	}

	return include, diags
}

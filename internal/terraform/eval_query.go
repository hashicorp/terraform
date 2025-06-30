// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// defaultQueryLimit is the default value for the "limit" argument in a list block.
const defaultQueryLimit = int64(100)

// newLimitEvaluator returns an evaluator for the limit expression within a list block.
func newLimitEvaluator(allowUnknown bool) *ExprEvaluator[cty.Type, int64] {
	return &ExprEvaluator[cty.Type, int64]{
		cType:        cty.Number,
		defaultValue: defaultQueryLimit,
		argName:      "limit",
		allowUnknown: allowUnknown,
		validateGoValue: func(expr hcl.Expression, val int64) tfdiags.Diagnostics {
			var diags tfdiags.Diagnostics
			if val < 0 {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid limit argument",
					Detail:   `The given "limit" argument value is unsuitable: must be greater than or equal to zero.`,
					Subject:  expr.Range().Ptr(),
				})
				return diags
			}
			return diags
		},
	}
}

// newIncludeRscEvaluator returns an evaluator for the include_resource expression.
func newIncludeRscEvaluator(allowUnknown bool) *ExprEvaluator[cty.Type, bool] {
	return &ExprEvaluator[cty.Type, bool]{
		cType:        cty.Bool,
		defaultValue: false,
		argName:      "include_resource",
		allowUnknown: allowUnknown,
	}
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

func evaluateImportIdExpression(expr hcl.Expression, ctx EvalContext, keyData instances.RepetitionData) (string, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// import blocks only exist in the root module, and must be evaluated in
	// that context.
	ctx = ctx.WithPath(addrs.RootModuleInstance)

	if expr == nil {
		return "", diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid import id argument",
			Detail:   "The import ID cannot be null.",
			Subject:  expr.Range().Ptr(),
		})
	}

	scope := ctx.EvaluationScope(nil, nil, keyData)
	importIdVal, evalDiags := scope.EvalExpr(expr, cty.String)
	diags = diags.Append(evalDiags)

	if importIdVal.IsNull() {
		return "", diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid import id argument",
			Detail:   "The import ID cannot be null.",
			Subject:  expr.Range().Ptr(),
		})
	}

	if !importIdVal.IsKnown() {
		return "", diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid import id argument",
			Detail:   `The import block "id" argument depends on resource attributes that cannot be determined until apply, so Terraform cannot plan to import this resource.`, // FIXME and what should I do about that?
			Subject:  expr.Range().Ptr(),
			//	Expression:
			//	EvalContext:
			Extra: diagnosticCausedByUnknown(true),
		})
	}

	var importId string
	err := gocty.FromCtyValue(importIdVal, &importId)
	if err != nil {
		return "", diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid import id argument",
			Detail:   fmt.Sprintf("The import ID value is unsuitable: %s.", err),
			Subject:  expr.Range().Ptr(),
		})
	}

	return importId, diags
}

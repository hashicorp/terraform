// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type evalContextExprEvaluator struct {
	ctx EvalContext
}

func (e evalContextExprEvaluator) EvaluateStringExpr(
	expr hcl.Expression,
) (cty.Value, tfdiags.Diagnostics) {
	return e.ctx.EvaluateExpr(expr, cty.String, nil)
}

func ResolveProvider(
	name string,
	expr *configs.ProviderRequirementExpr,
	ctx EvalContext,
) (*configs.RequiredProvider, tfdiags.Diagnostics) {
	rp, _, diags := configs.ResolveProviderRequirement(
		name,
		expr,
		evalContextExprEvaluator{ctx: ctx},
		configs.ProviderRequirementResolveOpts{DeferUnresolved: false},
	)
	return rp, diags
}

func ResolveProviderWithHCLContext(
	name string,
	expr *configs.ProviderRequirementExpr,
	ctx *hcl.EvalContext,
) (*configs.RequiredProvider, tfdiags.Diagnostics) {
	rp, _, diags := configs.ResolveProviderRequirement(
		name,
		expr,
		configs.HCLEvalExprEvaluator{Ctx: ctx},
		configs.ProviderRequirementResolveOpts{DeferUnresolved: false},
	)
	return rp, diags
}

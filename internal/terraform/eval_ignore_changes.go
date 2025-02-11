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
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// evaluateIgnoreChangesConditional is used to evaluate ignore_changes arguments that are conditional expressions
func evaluateIgnoreChangesConditional(expr hcl.Expression, ctx EvalContext, keyData instances.RepetitionData) ([]hcl.Traversal, tfdiags.Diagnostics) {
	conditional, ok := expr.(*hclsyntax.ConditionalExpr)
	if !ok {
		var diags tfdiags.Diagnostics
		diags = diags.Append(fmt.Errorf("ignore_changes value is not a conditional expression but is being processed as one; this is a bug in Terraform. Condition: %#v", expr))
		return []hcl.Traversal{}, diags
	}

	return newIgnoreChangesConditionalEvaluator(conditional, ctx, keyData).evaluate()
}

// IgnoreChangesConditional is the standard mechanism for interpreting an expression
// given for an "ignore_changes" argument in a lifecycle block.
func newIgnoreChangesConditionalEvaluator(expr *hclsyntax.ConditionalExpr, ctx EvalContext, keyData instances.RepetitionData) *ignoreChangesConditionalEvaluator {
	if ctx == nil {
		panic("nil EvalContext")
	}

	return &ignoreChangesConditionalEvaluator{
		ctx:     ctx,
		expr:    expr,
		keyData: keyData,
	}
}

type ignoreChangesConditionalEvaluator struct {
	ctx     EvalContext
	expr    *hclsyntax.ConditionalExpr
	keyData instances.RepetitionData

	// internal
	hclCtx *hcl.EvalContext
}

func (ev *ignoreChangesConditionalEvaluator) evaluate() ([]hcl.Traversal, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	val, d := ev.evaluateCondition()
	diags = diags.Append(d)
	if diags.HasErrors() {
		return []hcl.Traversal{}, diags
	}

	// We cannot handle unknowns in ignore_changes
	if !val.IsKnown() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "Invalid ignore_changes argument",
			Detail:      "The condition could not be evaluated at this time, a result will be known when this plan is applied.",
			Subject:     ev.expr.Condition.Range().Ptr(),
			Expression:  ev.expr.Condition,
			EvalContext: ev.hclCtx,
		})
		return []hcl.Traversal{}, diags
	}

	result, d := ev.evaluateResult(val)
	diags = diags.Append(d)
	if diags.HasErrors() {
		return []hcl.Traversal{}, diags
	}

	return result, diags
}

// evaluateCondition returns the value of the conditional expression's condition
func (ev *ignoreChangesConditionalEvaluator) evaluateCondition() (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var refs []*addrs.Reference

	r, d := langrefs.ReferencesInExpr(addrs.ParseRef, ev.expr.Condition)
	diags = diags.Append(d)
	refs = append(refs, r...)

	scope := ev.ctx.EvaluationScope(nil, nil, ev.keyData)

	hclCtx, moreDiags := scope.EvalContext(refs)
	diags = diags.Append(moreDiags)
	ev.hclCtx = hclCtx

	condition, hclDiags := ev.expr.Condition.Value(ev.hclCtx)
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		return cty.Value{}, diags
	}

	return condition, diags
}

// evaluateResult returns the traversals from the result returned from the conditional expression
func (ev *ignoreChangesConditionalEvaluator) evaluateResult(exprVal cty.Value) ([]hcl.Traversal, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var traversals []hcl.Traversal

	var result hclsyntax.Expression

	if exprVal == cty.UnknownVal(cty.Bool) {
		panic("unknown expression value")
	}
	if exprVal == cty.BoolVal(true) {
		result = ev.expr.TrueResult
	}
	if exprVal == cty.BoolVal(false) {
		result = ev.expr.FalseResult
	}

	switch result.(type) {
	case *hclsyntax.TupleConsExpr:
		if v, ok := result.(*hclsyntax.TupleConsExpr); ok {
			for _, e := range v.ExprList() {

				if exprIsNativeQuotedString(e) {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid ignore_changes argument",
						Detail: fmt.Sprintf(
							"An \"ignore_changes\" conditional expression should not contain quoted strings when listing field names.",
						),
						Subject:     ev.expr.Range().Ptr(),
						Expression:  ev.expr,
						EvalContext: ev.hclCtx,
					})
				}

				// Get a relative traversal for each entry in the field reference list
				traversal, d := hcl.RelTraversalForExpr(e)
				traversals = append(traversals, traversal)
				diags = diags.Append(d)
			}
		}
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid ignore_changes argument",
			Detail: fmt.Sprintf(
				"Unable to evaluate the '%s' result in an \"ignore_changes\" conditional expression. The conditional expression should contain two list/tuple values to be used after the conditional expression is evaluated.",
				exprVal.AsString(),
			),
			Subject:     ev.expr.Range().Ptr(),
			Expression:  ev.expr,
			EvalContext: ev.hclCtx,
		})
		return []hcl.Traversal{}, diags
	}

	return traversals, diags
}

// exprIsNativeQuotedString determines whether the given expression looks like
// it's a quoted string in the HCL native syntax.
//
// This should be used sparingly only for situations where our legacy HCL
// decoding would've expected a keyword or reference in quotes but our new
// decoding expects the keyword or reference to be provided directly as
// an identifier-based expression.
func exprIsNativeQuotedString(expr hcl.Expression) bool {
	_, ok := expr.(*hclsyntax.TemplateExpr)
	return ok
}

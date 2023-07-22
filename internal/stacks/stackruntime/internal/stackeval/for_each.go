package stackeval

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// evaluateForEachExpr deals with all of the for_each evaluation concerns
// that are common across all uses of for_each in all evaluation phases.
//
// The caller might still need to do some further validation or post-processing
// of the result for concerns that are specific to a particular phase or
// evaluation context.
func evaluateForEachExpr(ctx context.Context, expr hcl.Expression, phase EvalPhase, scope ExpressionScope) (ExprResultValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	result, moreDiags := EvalExprAndEvalContext(
		ctx, expr, phase, scope,
	)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return ExprResultValue{
			Value:       cty.DynamicVal,
			Expression:  expr,
			EvalContext: nil,
		}, diags
	}
	ty := result.Value.Type()

	const invalidForEachSummary = "Invalid for_each value"
	const invalidForEachDetail = "The for_each expression must produce either a map of any type or a set of strings. The keys of the map or the set elements will serve as unique identifiers for multiple instances of this embedded stack."
	switch {
	case ty.IsObjectType() || ty.IsMapType():
		// okay
	case ty.IsSetType():
		if !ty.ElementType().Equals(cty.String) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     invalidForEachSummary,
				Detail:      invalidForEachDetail,
				Subject:     result.Expression.Range().Ptr(),
				Expression:  result.Expression,
				EvalContext: result.EvalContext,
			})
			return DerivedExprResult(result, cty.DynamicVal), diags
		}
	default:
		if !ty.ElementType().Equals(cty.String) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     invalidForEachSummary,
				Detail:      invalidForEachDetail,
				Subject:     result.Expression.Range().Ptr(),
				Expression:  result.Expression,
				EvalContext: result.EvalContext,
			})
			return DerivedExprResult(result, cty.DynamicVal), diags
		}
	}
	if result.Value.IsNull() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     invalidForEachSummary,
			Detail:      "The for_each value must not be null.",
			Subject:     result.Expression.Range().Ptr(),
			Expression:  result.Expression,
			EvalContext: result.EvalContext,
		})
		return DerivedExprResult(result, cty.DynamicVal), diags
	}

	// Unknown and sensitive values are also typically disallowed, but
	// known-ness and sensitivity get decided dynamically based on data flow
	// and so we'll treat those as plan-time errors only.

	return result, diags
}

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
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
func evaluateCountExpression(expr hcl.Expression, ctx EvalContext) (int, tfdiags.Diagnostics) {
	countVal, diags := evaluateCountExpressionValue(expr, ctx)
	if !countVal.IsKnown() {
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
		})
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
			Detail:   `The given "count" argument value is unsuitable: negative numbers are not supported.`,
			Subject:  expr.Range().Ptr(),
		})
		return nullCount, diags
	}

	return countVal, diags
}

// fixResourceCountSetTransition is a helper function to fix up the state when a
// resource transitions its "count" from being set to unset or vice-versa,
// treating a 0-key and a no-key instance as aliases for one another across
// the transition.
//
// The correct time to call this function is in the DynamicExpand method for
// a node representing a resource, just after evaluating the count with
// evaluateCountExpression, and before any other analysis of the
// state such as orphan detection.
//
// This function calls methods on the given EvalContext to update the current
// state in-place, if necessary. It is a no-op if there is no count transition
// taking place.
//
// Since the state is modified in-place, this function must take a writer lock
// on the state. The caller must therefore not also be holding a state lock,
// or this function will block forever awaiting the lock.
func fixResourceCountSetTransition(ctx EvalContext, addr addrs.ConfigResource, countEnabled bool) {
	state := ctx.State()
	changed := state.MaybeFixUpResourceInstanceAddressForCount(addr, countEnabled)
	if changed {
		log.Printf("[TRACE] renamed first %s instance in transient state due to count argument change", addr)
	}
}

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// evaluateResourceForEachExpression interprets a "for_each" argument on a resource.
//
// Returns a cty.Value map, and diagnostics if necessary. It will return nil if
// the expression is nil, and is used to distinguish between an unset for_each and an
// empty map
func evaluateResourceForEachExpression(expr hcl.Expression, ctx EvalContext) (forEach map[string]cty.Value, diags tfdiags.Diagnostics) {
	forEachMap, known, diags := evaluateResourceForEachExpressionKnown(expr, ctx)
	if !known {
		// Currently this is a rather bad outcome from a UX standpoint, since we have
		// no real mechanism to deal with this situation and all we can do is produce
		// an error message.
		// FIXME: In future, implement a built-in mechanism for deferring changes that
		// can't yet be predicted, and use it to guide the user through several
		// plan/apply steps until the desired configuration is eventually reached.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid forEach argument",
			Detail:   `The "for_each" value depends on resource attributes that cannot be determined until apply, so Terraform cannot predict how many instances will be created. To work around this, use the -target argument to first apply only the resources that the for_each depends on.`,
		})
		panic("uh-oh")
	}
	return forEachMap, diags
}

// evaluateResourceForEachExpressionKnown is like evaluateResourceForEachExpression
// except that it handles an unknown result by returning an empty map and
// a known = false, rather than by reporting the unknown value as an error
// diagnostic.
func evaluateResourceForEachExpressionKnown(expr hcl.Expression, ctx EvalContext) (forEach map[string]cty.Value, known bool, diags tfdiags.Diagnostics) {
	if expr == nil {
		return nil, true, nil
	}

	forEachVal, forEachDiags := ctx.EvaluateExpr(expr, cty.DynamicPseudoType, nil)
	diags = diags.Append(forEachDiags)
	if diags.HasErrors() {
		return nil, true, diags
	}

	switch {
	case forEachVal.IsNull():
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid for_each argument",
			Detail:   `The given "for_each" argument value is unsuitable: the given "for_each" argument value is null. A map, or set of strings is allowed.`,
			Subject:  expr.Range().Ptr(),
		})
		return nil, true, diags
	case !forEachVal.IsKnown():
		return map[string]cty.Value{}, false, diags
	}

	if !forEachVal.CanIterateElements() || forEachVal.Type().IsListType() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid for_each argument",
			Detail:   fmt.Sprintf(`The given "for_each" argument value is unsuitable: the "for_each" argument must be a map, or set of strings, and you have provided a value of type %s.`, forEachVal.Type().FriendlyName()),
			Subject:  expr.Range().Ptr(),
		})
		return nil, true, diags
	}

	if forEachVal.Type().IsSetType() {
		if forEachVal.Type().ElementType() != cty.String {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid for_each set argument",
				Detail:   fmt.Sprintf(`The given "for_each" argument value is unsuitable: "for_each" supports maps and sets of strings, but you have provided a set containing type %s.`, forEachVal.Type().ElementType().FriendlyName()),
				Subject:  expr.Range().Ptr(),
			})
			return nil, true, diags
		}
	}

	// If the map is empty ({}), return an empty map, because cty will return nil when representing {} AsValueMap
	if forEachVal.LengthInt() == 0 {
		return map[string]cty.Value{}, true, diags
	}

	return forEachVal.AsValueMap(), true, nil
}

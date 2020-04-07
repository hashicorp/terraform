package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// evaluateForEachExpression interprets a "for_each" argument on a resource.
// As opposed to evaluateForEachExpressionKnwon, evaluateForEachExpression will
// return an error if the resulting value is unknown.
// The returned cty.Value will be either a cty.Set or cty.Map type suitable for
// for_each expansion.
func evaluateForEachExpression(expr hcl.Expression, ctx EvalContext) (cty.Value, tfdiags.Diagnostics) {
	forEach, diags := evaluateForEachExpressionKnown(expr, ctx)

	known := true
	switch {
	case forEach.Type().IsMapType():
		known = forEach.IsKnown()
	case forEach.Type().IsSetType():
		known = forEach.IsWhollyKnown()
	}

	if !known {
		// Attach a diag as we do with count, with the same downsides
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid for_each argument",
			Detail:   `The "for_each" value depends on resource attributes that cannot be determined until apply, so Terraform cannot predict how many instances will be created. To work around this, use the -target argument to first apply only the resources that the for_each depends on.`,
			Subject:  expr.Range().Ptr(),
		})
		forEach = cty.MapValEmpty(cty.DynamicPseudoType)
	}
	return forEach, diags
}

func evaluateForEachExpressionKnown(expr hcl.Expression, ctx EvalContext) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// This is still called unconditionally to attempt to extract key data in
	// some places
	if expr == nil {
		return cty.MapValEmpty(cty.DynamicPseudoType), diags
	}

	forEachVal, forEachDiags := ctx.EvaluateExpr(expr, cty.DynamicPseudoType, nil)
	diags = diags.Append(forEachDiags)
	if diags.HasErrors() {
		return forEachVal, diags
	}

	ty := forEachVal.Type()

	switch {
	case forEachVal.IsNull():
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid for_each argument",
			Detail:   `The given "for_each" argument value is unsuitable: the given "for_each" argument value is null. A map, or set of strings is allowed.`,
			Subject:  expr.Range().Ptr(),
		})
		return forEachVal, diags

	case !ty.IsMapType() &&
		!ty.IsObjectType() &&
		!ty.IsSetType():
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid for_each argument",
			Detail:   fmt.Sprintf(`The given "for_each" argument value is unsuitable: the "for_each" argument must be a map, or set of strings, and you have provided a value of type %s.`, ty.FriendlyName()),
			Subject:  expr.Range().Ptr(),
		})
		return forEachVal, diags

	case !forEachVal.IsKnown():
		return forEachVal, diags

	}

	// validate the set values
	if ty.IsSetType() {
		// an empty set may have a dynamic type, we we need to ensure that the
		// correct type is returned here.
		if forEachVal.LengthInt() == 0 {
			return cty.SetValEmpty(cty.String), diags
		}

		if ty.ElementType() != cty.String {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid for_each set argument",
				Detail:   fmt.Sprintf(`The given "for_each" argument value is unsuitable: "for_each" supports maps and sets of strings, but you have provided a set containing type %s.`, ty.ElementType().FriendlyName()),
				Subject:  expr.Range().Ptr(),
			})
			return forEachVal, diags
		}
		// A set of strings may contain null, which makes it impossible to
		// convert to a map, so we must return an error
		it := forEachVal.ElementIterator()
		for it.Next() {
			item, _ := it.Element()
			if item.IsNull() {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid for_each set argument",
					Detail:   fmt.Sprintf(`The given "for_each" argument value is unsuitable: "for_each" sets must not contain null values.`),
					Subject:  expr.Range().Ptr(),
				})
				return forEachVal, diags
			}
		}
	}

	return forEachVal, diags
}

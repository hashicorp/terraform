package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// evaluateForEachExpression is our standard mechanism for interpreting an
// expression given for a "for_each" argument on a resource or a module. This
// should be called during expansion in order to determine the final keys and
// values.
//
// evaluateForEachExpression differs from evaluateForEachExpressionValue by
// returning an error if the count value is not known, and converting the
// cty.Value to a map[string]cty.Value for compatibility with other calls.
func evaluateForEachExpression(expr hcl.Expression, ctx EvalContext) (forEach map[string]cty.Value, diags tfdiags.Diagnostics) {
	forEachVal, diags := evaluateForEachExpressionValue(expr, ctx)
	if !forEachVal.IsKnown() {
		// Attach a diag as we do with count, with the same downsides
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid for_each argument",
			Detail:   `The "for_each" value depends on resource attributes that cannot be determined until apply, so Terraform cannot predict how many instances will be created. To work around this, use the -target argument to first apply only the resources that the for_each depends on.`,
			Subject:  expr.Range().Ptr(),
		})
	}

	if forEachVal.IsNull() || !forEachVal.IsKnown() || forEachVal.LengthInt() == 0 {
		// we check length, because an empty set return a nil map
		return map[string]cty.Value{}, diags
	}

	return forEachVal.AsValueMap(), diags
}

// evaluateForEachExpressionValue is like evaluateForEachExpression
// except that it returns a cty.Value map or set which can be unknown.
func evaluateForEachExpressionValue(expr hcl.Expression, ctx EvalContext) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	nullMap := cty.NullVal(cty.Map(cty.DynamicPseudoType))

	if expr == nil {
		return nullMap, diags
	}

	forEachVal, forEachDiags := ctx.EvaluateExpr(expr, cty.DynamicPseudoType, nil)
	diags = diags.Append(forEachDiags)
	if diags.HasErrors() {
		return nullMap, diags
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
		return nullMap, diags
	case !forEachVal.IsKnown():
		// ensure that we have a map, and not a DynamicValue
		return cty.UnknownVal(cty.Map(cty.DynamicPseudoType)), diags

	case !(ty.IsMapType() || ty.IsSetType() || ty.IsObjectType()):
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid for_each argument",
			Detail:   fmt.Sprintf(`The given "for_each" argument value is unsuitable: the "for_each" argument must be a map, or set of strings, and you have provided a value of type %s.`, ty.FriendlyName()),
			Subject:  expr.Range().Ptr(),
		})
		return nullMap, diags

	case forEachVal.LengthInt() == 0:
		// If the map is empty ({}), return an empty map, because cty will
		// return nil when representing {} AsValueMap. This also covers an empty
		// set (toset([]))
		return forEachVal, diags
	}

	if ty.IsSetType() {
		// since we can't use a set values that are unknown, we treat the
		// entire set as unknown
		if !forEachVal.IsWhollyKnown() {
			return cty.UnknownVal(ty), diags
		}

		if ty.ElementType() != cty.String {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid for_each set argument",
				Detail:   fmt.Sprintf(`The given "for_each" argument value is unsuitable: "for_each" supports maps and sets of strings, but you have provided a set containing type %s.`, forEachVal.Type().ElementType().FriendlyName()),
				Subject:  expr.Range().Ptr(),
			})
			return cty.NullVal(ty), diags
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
				return cty.NullVal(ty), diags
			}
		}
	}

	return forEachVal, nil
}

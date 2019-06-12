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
	if expr == nil {
		return nil, nil
	}

	forEachVal, forEachDiags := ctx.EvaluateExpr(expr, cty.DynamicPseudoType, nil)
	diags = diags.Append(forEachDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	// No-op for dynamic types, so that these pass validation, but are then populated at apply
	if forEachVal.Type() == cty.DynamicPseudoType {
		return nil, diags
	}

	if forEachVal.IsNull() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid for_each argument",
			Detail:   `The given "for_each" argument value is unsuitable: the given "for_each" argument value is null. A map, or set of strings is allowed.`,
			Subject:  expr.Range().Ptr(),
		})
		return nil, diags
	}

	if !forEachVal.CanIterateElements() || forEachVal.Type().IsListType() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid for_each argument",
			Detail:   fmt.Sprintf(`The given "for_each" argument value is unsuitable: the "for_each" argument must be a map, or set of strings, and you have provided a value of type %s.`, forEachVal.Type().FriendlyName()),
			Subject:  expr.Range().Ptr(),
		})
		return nil, diags
	}

	if forEachVal.Type().IsSetType() {
		if forEachVal.Type().ElementType() != cty.String {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid for_each set argument",
				Detail:   fmt.Sprintf(`The given "for_each" argument value is unsuitable: "for_each" supports maps and sets of strings, but you have provided a set containing type %s.`, forEachVal.Type().ElementType().FriendlyName()),
				Subject:  expr.Range().Ptr(),
			})
			return nil, diags
		}
	}

	// If the map is empty ({}), return an empty map, because cty will return nil when representing {} AsValueMap
	// Also return an empty map if the value is not known -- as this function
	// is used to check if the for_each value is valid as well as to apply it, the empty
	// map will later be filled in.
	if !forEachVal.IsKnown() || forEachVal.LengthInt() == 0 {
		return map[string]cty.Value{}, diags
	}

	return forEachVal.AsValueMap(), nil
}

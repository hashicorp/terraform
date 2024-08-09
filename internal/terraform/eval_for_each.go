// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// evaluateForEachExpression differs from evaluateForEachExpressionValue by
// returning an error if the count value is not known, and converting the
// cty.Value to a map[string]cty.Value for compatibility with other calls.
func evaluateForEachExpression(expr hcl.Expression, ctx EvalContext, allowUnknown bool) (forEach map[string]cty.Value, known bool, diags tfdiags.Diagnostics) {
	return newForEachEvaluator(expr, ctx, allowUnknown).ResourceValue()
}

// forEachEvaluator is the standard mechanism for interpreting an expression
// given for a "for_each" argument on a resource, module, or import.
func newForEachEvaluator(expr hcl.Expression, ctx EvalContext, allowUnknown bool) *forEachEvaluator {
	if ctx == nil {
		panic("nil EvalContext")
	}

	return &forEachEvaluator{
		ctx:          ctx,
		expr:         expr,
		allowUnknown: allowUnknown,
	}
}

// forEachEvaluator is responsible for evaluating for_each expressions, using
// different rules depending on the desired context.
type forEachEvaluator struct {
	// We bundle this functionality into a structure, because internal
	// validation requires not only the resulting value, but also the original
	// expression and the hcl EvalContext to build the corresponding
	// diagnostic. Every method's dependency on all the evaluation pieces
	// otherwise prevents refactoring and we end up with a single giant
	// function.
	ctx  EvalContext
	expr hcl.Expression

	// TEMP: If allowUnknown is set then we skip the usual restriction that
	// unknown values are not allowed in for_each. A caller that sets this
	// must therefore be ready to deal with the result being unknown.
	// This will eventually become the default behavior, once we've updated
	// the rest of this package to handle that situation in a reasonable way.
	allowUnknown bool

	// internal
	hclCtx *hcl.EvalContext
}

// ResourceForEachValue returns a known for_each map[string]cty.Value
// appropriate for use within resource expansion.
func (ev *forEachEvaluator) ResourceValue() (map[string]cty.Value, bool, tfdiags.Diagnostics) {
	res := map[string]cty.Value{}

	// no expression always results in an empty map
	if ev.expr == nil {
		return res, true, nil
	}

	forEachVal, diags := ev.Value()
	if diags.HasErrors() {
		return res, false, diags
	}

	// ensure our value is known for use in resource expansion
	unknownDiags := ev.ensureKnownForResource(forEachVal)
	if unknownDiags.HasErrors() {
		if !ev.allowUnknown {
			diags = diags.Append(unknownDiags)
		}
		return res, false, diags
	}

	// validate the for_each value for use in resource expansion
	diags = diags.Append(ev.validateResource(forEachVal))
	if diags.HasErrors() {
		return res, false, diags
	}

	if forEachVal.IsNull() || !forEachVal.IsKnown() || markSafeLengthInt(forEachVal) == 0 {
		// we check length, because an empty set returns a nil map which will panic below
		return res, true, diags
	}

	if _, marks := forEachVal.Unmark(); len(marks) != 0 {
		// Should not get here, because validateResource above should have
		// rejected values that are marked. If we do get here then it's
		// likely that we've added a new kind of mark that validateResource
		// doesn't know about yet, and so we'll need to decide how for_each
		// should react to that new mark.
		diags = diags.Append(fmt.Errorf("for_each value is marked with %#v despite earlier validation; this is a bug in Terraform", marks))
		return res, false, diags
	}
	res = forEachVal.AsValueMap()
	return res, true, diags
}

// ImportValue returns the for_each map for use within an import block,
// enumerated as individual instances.RepetitionData values.
func (ev *forEachEvaluator) ImportValues() ([]instances.RepetitionData, bool, tfdiags.Diagnostics) {
	var res []instances.RepetitionData
	if ev.expr == nil {
		return res, true, nil
	}

	forEachVal, diags := ev.Value()
	if diags.HasErrors() {
		return res, false, diags
	}

	// ensure our value is known for use in resource expansion
	unknownDiags := diags.Append(ev.ensureKnownForImport(forEachVal))
	if unknownDiags.HasErrors() {
		if !ev.allowUnknown {
			diags = diags.Append(unknownDiags)
		}
		return res, false, diags
	}

	if forEachVal.IsNull() {
		return res, true, diags
	}

	val, marks := forEachVal.Unmark()

	if !val.CanIterateElements() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "Invalid for_each argument",
			Detail:      "The \"for_each\" expression must be a collection.",
			Subject:     ev.expr.Range().Ptr(),
			Expression:  ev.expr,
			EvalContext: ev.hclCtx,
		})
		return res, false, diags
	}

	it := val.ElementIterator()
	for it.Next() {
		k, v := it.Element()
		res = append(res, instances.RepetitionData{
			EachKey:   k,
			EachValue: v.WithMarks(marks),
		})

	}

	return res, true, diags
}

// Value returns the raw cty.Value evaluated from the given for_each expression
func (ev *forEachEvaluator) Value() (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if ev.expr == nil {
		// a nil expression always results in a null value
		return cty.NullVal(cty.Map(cty.DynamicPseudoType)), nil
	}

	refs, moreDiags := langrefs.ReferencesInExpr(addrs.ParseRef, ev.expr)
	diags = diags.Append(moreDiags)
	scope := ev.ctx.EvaluationScope(nil, nil, EvalDataForNoInstanceKey)
	if scope != nil {
		ev.hclCtx, moreDiags = scope.EvalContext(refs)
	} else {
		// This shouldn't happen in real code, but it can unfortunately arise
		// in unit tests due to incompletely-implemented mocks. :(
		ev.hclCtx = &hcl.EvalContext{}
	}

	diags = diags.Append(moreDiags)
	if diags.HasErrors() { // Can't continue if we don't even have a valid scope
		return cty.DynamicVal, diags
	}

	forEachVal, forEachDiags := ev.expr.Value(ev.hclCtx)
	diags = diags.Append(forEachDiags)

	return forEachVal, diags
}

// ensureKnownForImport checks that the value is entirely known for use within
// import expansion.
func (ev *forEachEvaluator) ensureKnownForImport(forEachVal cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if !forEachVal.IsWhollyKnown() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "Invalid for_each argument",
			Detail:      "The \"for_each\" expression includes values derived from other resource attributes that cannot be determined until apply, and so Terraform cannot determine the full set of values that might be used to import this resource.",
			Subject:     ev.expr.Range().Ptr(),
			Expression:  ev.expr,
			EvalContext: ev.hclCtx,
			Extra:       diagnosticCausedByUnknown(true),
		})
	}
	return diags
}

// ensureKnownForResource checks that the value is known within the rules of
// resource and module expansion.
func (ev *forEachEvaluator) ensureKnownForResource(forEachVal cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	ty := forEachVal.Type()
	const errInvalidUnknownDetailMap = "The \"for_each\" map includes keys derived from resource attributes that cannot be determined until apply, and so Terraform cannot determine the full set of keys that will identify the instances of this resource.\n\nWhen working with unknown values in for_each, it's better to define the map keys statically in your configuration and place apply-time results only in the map values.\n\nAlternatively, you could use the -target planning option to first apply only the resources that the for_each value depends on, and then apply a second time to fully converge."
	const errInvalidUnknownDetailSet = "The \"for_each\" set includes values derived from resource attributes that cannot be determined until apply, and so Terraform cannot determine the full set of keys that will identify the instances of this resource.\n\nWhen working with unknown values in for_each, it's better to use a map value where the keys are defined statically in your configuration and where only the values contain apply-time results.\n\nAlternatively, you could use the -target planning option to first apply only the resources that the for_each value depends on, and then apply a second time to fully converge."

	if !forEachVal.IsKnown() {
		var detailMsg string
		switch {
		case ty.IsSetType():
			detailMsg = errInvalidUnknownDetailSet
		default:
			detailMsg = errInvalidUnknownDetailMap
		}

		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "Invalid for_each argument",
			Detail:      detailMsg,
			Subject:     ev.expr.Range().Ptr(),
			Expression:  ev.expr,
			EvalContext: ev.hclCtx,
			Extra:       diagnosticCausedByUnknown(true),
		})
		return diags
	}

	if ty.IsSetType() && !forEachVal.IsWhollyKnown() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "Invalid for_each argument",
			Detail:      errInvalidUnknownDetailSet,
			Subject:     ev.expr.Range().Ptr(),
			Expression:  ev.expr,
			EvalContext: ev.hclCtx,
			Extra:       diagnosticCausedByUnknown(true),
		})
	}
	return diags
}

// ValidateResourceValue is used from validation walks to verify the validity
// of the resource for_Each expression, while still allowing for unknown
// values.
func (ev *forEachEvaluator) ValidateResourceValue() tfdiags.Diagnostics {
	val, diags := ev.Value()
	if diags.HasErrors() {
		return diags
	}

	return diags.Append(ev.validateResource(val))
}

// validateResource validates the type and values of the forEachVal, while
// still allowing unknown values for use within the validation walk.
func (ev *forEachEvaluator) validateResource(forEachVal cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// Sensitive values are not allowed because otherwise the sensitive keys
	// would get exposed as part of the instance addresses.
	if forEachVal.HasMark(marks.Sensitive) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "Invalid for_each argument",
			Detail:      "Sensitive values, or values derived from sensitive values, cannot be used as for_each arguments. If used, the sensitive value could be exposed as a resource instance key.",
			Subject:     ev.expr.Range().Ptr(),
			Expression:  ev.expr,
			EvalContext: ev.hclCtx,
			Extra:       diagnosticCausedBySensitive(true),
		})
	}
	// Ephemeral values are not allowed because instance keys persist from
	// plan to apply and between plan/apply rounds, whereas ephemeral values
	// do not.
	if forEachVal.HasMark(marks.Ephemeral) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "Invalid for_each argument",
			Detail:      `The given "for_each" value is derived from an ephemeral value, which means that Terraform cannot persist it between plan/apply rounds. Use only non-ephemeral values to specify a resource's instance keys.`,
			Subject:     ev.expr.Range().Ptr(),
			Expression:  ev.expr,
			EvalContext: ev.hclCtx,
			Extra:       diagnosticCausedByEphemeral(true),
		})
	}

	if diags.HasErrors() {
		return diags
	}
	ty := forEachVal.Type()

	switch {
	case forEachVal.IsNull():
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "Invalid for_each argument",
			Detail:      `The given "for_each" argument value is unsuitable: the given "for_each" argument value is null. A map, or set of strings is allowed.`,
			Subject:     ev.expr.Range().Ptr(),
			Expression:  ev.expr,
			EvalContext: ev.hclCtx,
		})
		return diags

	case forEachVal.Type() == cty.DynamicPseudoType:
		// We may not have any type information if this is during validation,
		// so we need to return early. During plan this can't happen because we
		// validate for unknowns first.
		return diags

	case !(ty.IsMapType() || ty.IsSetType() || ty.IsObjectType()):
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "Invalid for_each argument",
			Detail:      fmt.Sprintf(`The given "for_each" argument value is unsuitable: the "for_each" argument must be a map, or set of strings, and you have provided a value of type %s.`, ty.FriendlyName()),
			Subject:     ev.expr.Range().Ptr(),
			Expression:  ev.expr,
			EvalContext: ev.hclCtx,
		})
		return diags

	case !forEachVal.IsKnown():
		return diags

	case markSafeLengthInt(forEachVal) == 0:
		// If the map is empty ({}), return an empty map, because cty will
		// return nil when representing {} AsValueMap. This also covers an empty
		// set (toset([]))
		return diags
	}

	if ty.IsSetType() {
		// since we can't use a set values that are unknown, we treat the
		// entire set as unknown
		if !forEachVal.IsWhollyKnown() {
			return diags
		}

		if ty.ElementType() != cty.String {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Invalid for_each set argument",
				Detail:      fmt.Sprintf(`The given "for_each" argument value is unsuitable: "for_each" supports maps and sets of strings, but you have provided a set containing type %s.`, forEachVal.Type().ElementType().FriendlyName()),
				Subject:     ev.expr.Range().Ptr(),
				Expression:  ev.expr,
				EvalContext: ev.hclCtx,
			})
			return diags
		}

		// A set of strings may contain null, which makes it impossible to
		// convert to a map, so we must return an error
		it := forEachVal.ElementIterator()
		for it.Next() {
			item, _ := it.Element()
			if item.IsNull() {
				diags = diags.Append(&hcl.Diagnostic{
					Severity:    hcl.DiagError,
					Summary:     "Invalid for_each set argument",
					Detail:      `The given "for_each" argument value is unsuitable: "for_each" sets must not contain null values.`,
					Subject:     ev.expr.Range().Ptr(),
					Expression:  ev.expr,
					EvalContext: ev.hclCtx,
				})
				return diags
			}
		}
	}

	return diags
}

// markSafeLengthInt allows calling LengthInt on marked values safely
func markSafeLengthInt(val cty.Value) int {
	v, _ := val.UnmarkDeep()
	return v.LengthInt()
}

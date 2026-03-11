// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type instancesResult[T any] struct {
	insts   map[addrs.InstanceKey]T
	unknown bool
}

// evaluateForEachExpr deals with all of the for_each evaluation concerns
// that are common across all uses of for_each in all evaluation phases.
//
// The caller might still need to do some further validation or post-processing
// of the result for concerns that are specific to a particular phase or
// evaluation context.
func evaluateForEachExpr(ctx context.Context, expr hcl.Expression, phase EvalPhase, scope ExpressionScope, callerDiagName string) (ExprResultValue, tfdiags.Diagnostics) {
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
	invalidForEachDetail := fmt.Sprintf("The for_each expression must produce either a map of any type or a set of strings. The keys of the map or the set elements will serve as unique identifiers for multiple instances of this %s.", callerDiagName)
	const sensitiveForEachDetail = "Sensitive values, or values derived from sensitive values, cannot be used as for_each arguments. If used, the sensitive value could be exposed as a resource instance key."
	switch {
	case result.Value.HasMark(marks.Sensitive):
		// Sensitive values are not allowed as for_each arguments because
		// they could be exposed as resource instance keys.
		// TODO: This should have Extra: tdiagnosticCausedBySensitive(true),
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     invalidForEachSummary,
			Detail:      sensitiveForEachDetail,
			Subject:     result.Expression.Range().Ptr(),
			Expression:  result.Expression,
			EvalContext: result.EvalContext,
		})
		return result, diags

	case result.Value.IsNull():
		// we don't alllow null values
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     invalidForEachSummary,
			Detail:      fmt.Sprintf("%s The for_each expression produced a null value.", invalidForEachDetail),
			Subject:     result.Expression.Range().Ptr(),
			Expression:  result.Expression,
			EvalContext: result.EvalContext,
		})
		return DerivedExprResult(result, cty.DynamicVal), diags

	case ty.IsObjectType() || ty.IsMapType():
		// okay

	case ty.IsSetType():
		// since we can't use a set values that are unknown, we treat the
		// entire set as unknown
		if !result.Value.IsWhollyKnown() {
			return result, diags
		}

		if markSafeLengthInt(result.Value) == 0 {
			// we are okay with an empty set
			return result, diags
		}

		if !ty.ElementType().Equals(cty.String) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     invalidForEachSummary,
				Detail:      fmt.Sprintf(`%s "for_each" supports maps and sets of strings, but you have provided a set containing type %s.`, invalidForEachDetail, ty.ElementType().FriendlyName()),
				Subject:     result.Expression.Range().Ptr(),
				Expression:  result.Expression,
				EvalContext: result.EvalContext,
			})
			return DerivedExprResult(result, cty.DynamicVal), diags
		}

		// Check if one of the values in the set is null
		for k, v := range result.Value.AsValueSet().Values() {
			if v.IsNull() {
				diags = diags.Append(&hcl.Diagnostic{
					Severity:    hcl.DiagError,
					Summary:     invalidForEachSummary,
					Detail:      fmt.Sprintf("%s The for_each value must not contain null elements, but the element at index %d was null.", invalidForEachDetail, k),
					Subject:     result.Expression.Range().Ptr(),
					Expression:  result.Expression,
					EvalContext: result.EvalContext,
				})
			}
		}

	case !result.Value.IsWhollyKnown() && ty.HasDynamicTypes():
		// If the value is unknown and has dynamic types, we can't
		// determine if it's a valid for_each value, so we'll just
		// return the unknown value.
		return result, diags

	default:
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

	// Sensitive values are also typically disallowed, but sensitivity gets
	// decided dynamically based on data flow and so we'll treat those as
	// plan-time errors, to be handled by the caller.

	return result, diags
}

// instancesMap constructs a map of instances of some expandable object,
// based on its for_each value or on the absence of such a value.
//
// If maybeForEachVal is [cty.NilVal] then the result is always a
// single-element map with an `addrs.NoKey` instance.
//
// If maybeForEachVal is non-nil then it must be a non-error result from
// an earlier call to [evaluateForEachExpr] which analyzed the given for_each
// expression. If the value is unknown then the result will be nil. Otherwise,
// the result is guaranteed to be a non-nil map with the same number of elements
// as the given for_each collection/structure.
//
// If maybeForEach value is non-nil but not a valid value produced by
// [evaluateForEachExpr] then the behavior is unpredictable, including the
// possibility of a panic.
func instancesMap[T any](maybeForEachVal cty.Value, makeInst func(addrs.InstanceKey, instances.RepetitionData) T) instancesResult[T] {
	switch {
	case maybeForEachVal == cty.NilVal:
		// No for_each expression at all, then. We have exactly one instance
		// without an instance key and with no repetition data.
		return instancesResult[T]{noForEachInstancesMap(makeInst), false}

	case !maybeForEachVal.IsKnown():
		// This is temporary to gradually rollout support for unknown for_each values
		return instancesResult[T]{nil, true}

	default:
		// Otherwise we should be able to assume the value is valid per the
		// definition of [evaluateForEachExpr]. The following will panic if
		// that other function doesn't satisfy its documented contract;
		// if that happens, prefer to correct the either that function or
		// its caller rather than adding further complexity here.

		// NOTE: We MUST return a non-nil map from every return path under
		// this case, even if there are zero elements in it, because a nil map
		// represents an _invalid_ for_each expression (handled above).
		// forEachInstancesMap guarantees to never return a nil map.
		return instancesResult[T]{forEachInstancesMap(maybeForEachVal, makeInst), false}

	}
}

// forEachInstanceKeys takes a value previously returned by
// [evaluateForEachExpr] and produces a map where each element maps from an
// instance key to a corresponding object decided by the givenc callback
// function.
//
// The result is guaranteed to be a non-nil map, even if the given value
// produces zero instances, because some callers use a nil map to represent
// the situation where the for_each value is too invalid to construct any
// map at all.
//
// This function is only designed to deal with valid (non-error) results from
// [evaluateForEachExpr] and so might panic if given other values.
func forEachInstancesMap[T any](forEachVal cty.Value, makeInst func(addrs.InstanceKey, instances.RepetitionData) T) map[addrs.InstanceKey]T {
	ty := forEachVal.Type()
	switch {
	case ty.IsObjectType() || ty.IsMapType():
		elems := forEachVal.AsValueMap()
		ret := make(map[addrs.InstanceKey]T, len(elems))
		for k, v := range elems {
			ik := addrs.StringKey(k)
			ret[ik] = makeInst(ik, instances.RepetitionData{
				EachKey:   cty.StringVal(k),
				EachValue: v,
			})
		}
		return ret

	case ty.IsSetType():
		if markSafeLengthInt(forEachVal) == 0 {
			// Zero-length for_each, so we have no instances.
			return make(map[addrs.InstanceKey]T)
		}

		// evaluateForEachExpr should have already guaranteed us a set of
		// strings, but we'll check again here just so we can panic more
		// intellgibly if that function is buggy.
		if ty.ElementType() != cty.String {
			panic(fmt.Sprintf("invalid forEachVal %#v", forEachVal))
		}

		elems := forEachVal.AsValueSlice()
		ret := make(map[addrs.InstanceKey]T, len(elems))
		for _, sv := range elems {
			k := addrs.StringKey(sv.AsString())
			ret[k] = makeInst(k, instances.RepetitionData{
				EachKey:   sv,
				EachValue: sv,
			})
		}
		return ret

	default:
		panic(fmt.Sprintf("invalid forEachVal %#v", forEachVal))
	}
}

func noForEachInstancesMap[T any](makeInst func(addrs.InstanceKey, instances.RepetitionData) T) map[addrs.InstanceKey]T {
	return map[addrs.InstanceKey]T{
		addrs.NoKey: makeInst(addrs.NoKey, instances.RepetitionData{
			// no repetition symbols available in this case
		}),
	}
}

// markSafeLengthInt allows calling LengthInt on marked values safely
func markSafeLengthInt(val cty.Value) int {
	v, _ := val.UnmarkDeep()
	return v.LengthInt()
}

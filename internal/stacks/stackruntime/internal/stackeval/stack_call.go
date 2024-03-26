// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StackCall represents a "stack" block in a stack configuration after
// its containing stacks have been expanded into stack instances.
type StackCall struct {
	addr stackaddrs.AbsStackCall

	main *Main

	forEachValue perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
	instances    perEvalPhase[promising.Once[withDiagnostics[map[addrs.InstanceKey]*StackCallInstance]]]
}

var _ Plannable = (*StackCall)(nil)
var _ Referenceable = (*StackCall)(nil)

func newStackCall(main *Main, addr stackaddrs.AbsStackCall) *StackCall {
	return &StackCall{
		addr: addr,
		main: main,
	}
}

func (c *StackCall) Addr() stackaddrs.AbsStackCall {
	return c.addr
}

func (c *StackCall) Config(ctx context.Context) *StackCallConfig {
	configAddr := stackaddrs.ConfigForAbs(c.addr)
	return c.main.StackCallConfig(ctx, configAddr)
}

func (c *StackCall) Caller(ctx context.Context) *Stack {
	callerAddr := c.Addr().Stack
	// Unchecked because StackCall instances only get constructed from
	// Stack objects, and so our address is derived from there.
	return c.main.StackUnchecked(ctx, callerAddr)
}

func (c *StackCall) Declaration(ctx context.Context) *stackconfig.EmbeddedStack {
	return c.Config(ctx).Declaration(ctx)
}

// ForEachValue returns the result of evaluating the "for_each" expression
// for this stack call, with the following exceptions:
//   - If the stack call doesn't use "for_each" at all, returns [cty.NilVal].
//   - If the for_each expression is present but too invalid to evaluate,
//     returns [cty.DynamicVal] to represent that the for_each value cannot
//     be determined.
//
// A present and valid "for_each" expression produces a result that's
// guaranteed to be:
// - Either a set of strings, a map of any element type, or an object type
// - Known and not null (only the top-level value)
// - Not sensitive (only the top-level value)
func (c *StackCall) ForEachValue(ctx context.Context, phase EvalPhase) cty.Value {
	ret, _ := c.CheckForEachValue(ctx, phase)
	return ret
}

// CheckForEachValue evaluates the "for_each" expression if present, validates
// that its value is valid, and then returns that value.
//
// If this call does not use "for_each" then this immediately returns cty.NilVal
// representing the absense of the value.
//
// If the diagnostics does not include errors and the result is not cty.NilVal
// then callers can assume that the result value will be:
// - Either a set of strings, a map of any element type, or an object type
// - Known and not null (except for nested map/object element values)
// - Not sensitive (only the top-level value)
//
// If the diagnostics _does_ include errors then the result might be
// [cty.DynamicVal], which represents that the for_each expression was so invalid
// that we cannot know the for_each value.
func (c *StackCall) CheckForEachValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	val, diags := doOnceWithDiags(
		ctx, c.forEachValue.For(phase), c.main,
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics
			cfg := c.Declaration(ctx)

			switch {

			case cfg.ForEach != nil:
				result, moreDiags := evaluateForEachExpr(ctx, cfg.ForEach, phase, c.Caller(ctx), "stack")
				diags = diags.Append(moreDiags)
				if diags.HasErrors() {
					return cty.DynamicVal, diags
				}

				if !result.Value.IsKnown() {
					// FIXME: We should somehow allow this and emit a
					// "deferred change" representing all of the as-yet-unknown
					// instances of this call and everything beneath it.
					diags = diags.Append(result.Diagnostic(
						tfdiags.Error,
						"Invalid for_each value",
						"The for_each value must not be derived from values that will be determined only during the apply phase.",
					))
				}

				return result.Value, diags

			default:
				// This stack config doesn't use for_each at all
				return cty.NilVal, diags
			}
		},
	)
	if val == cty.NilVal && diags.HasErrors() {
		// We use cty.DynamicVal as the placeholder for an invalid for_each,
		// to represent "unknown for_each value" as distinct from "no for_each
		// expression at all".
		val = cty.DynamicVal
	}
	return val, diags
}

// Instances returns all of the instances of the call known to be declared
// by the configuration.
//
// Calcluating this involves evaluating the call's for_each expression if any,
// and so this call may block on evaluation of other objects in the
// configuration.
//
// If the configuration has an invalid definition of the instances then the
// result will be nil. Callers that need to distinguish between invalid
// definitions and valid definitions of zero instances can rely on the
// result being a non-nil zero-length map in the latter case.
//
// This function doesn't return any diagnostics describing ways in which the
// for_each expression is invalid because we assume that the main plan walk
// will visit the stack call directly and ask it to check itself, and that
// call will be the one responsible for returning any diagnostics.
func (c *StackCall) Instances(ctx context.Context, phase EvalPhase) map[addrs.InstanceKey]*StackCallInstance {
	ret, _ := c.CheckInstances(ctx, phase)
	return ret
}

func (c *StackCall) CheckInstances(ctx context.Context, phase EvalPhase) (map[addrs.InstanceKey]*StackCallInstance, tfdiags.Diagnostics) {
	return doOnceWithDiags(
		ctx, c.instances.For(phase), c.main,
		func(ctx context.Context) (map[addrs.InstanceKey]*StackCallInstance, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics
			forEachVal := c.ForEachValue(ctx, phase)

			return instancesMap(forEachVal, func(ik addrs.InstanceKey, rd instances.RepetitionData) *StackCallInstance {
				return newStackCallInstance(c, ik, rd)
			}), diags
		},
	)
}

func (c *StackCall) ResultValue(ctx context.Context, phase EvalPhase) cty.Value {
	decl := c.Declaration(ctx)
	insts := c.Instances(ctx, phase)
	childResultType := c.Config(ctx).CalleeConfig(ctx).ResultType(ctx)

	switch {
	case decl.ForEach != nil:
		if insts == nil {
			// If we don't even know what instances we have then all we can
			// say is that our result ought to be a map of an object type
			// constructed from the child stack's output values.
			return cty.UnknownVal(cty.Map(childResultType))
		}

		// We expect that the instances all have string keys, which will
		// become the keys of a map that we're returning.
		elems := make(map[string]cty.Value, len(insts))
		for instKey, inst := range insts {
			k, ok := instKey.(addrs.StringKey)
			if !ok {
				panic(fmt.Sprintf("stack call with for_each has invalid instance key of type %T", instKey))
			}
			elems[string(k)] = inst.CalledStack(ctx).ResultValue(ctx, phase)
		}
		if len(elems) == 0 {
			return cty.MapValEmpty(childResultType)
		}
		return cty.MapVal(elems)

	default:
		if insts == nil {
			// If we don't even know what instances we have then all we can
			// say is that our result ought to have an object type
			// constructed from the child stack's output values.
			return cty.UnknownVal(childResultType)
		}
		if len(insts) != 1 {
			// Should not happen: we should have exactly one instance with addrs.NoKey
			panic("single-instance stack call does not have exactly one instance")
		}
		inst, ok := insts[addrs.NoKey]
		if !ok {
			panic("single-instance stack call does not have an addrs.NoKey instance")
		}

		return inst.CalledStack(ctx).ResultValue(ctx, phase)
	}
}

// ExprReferenceValue implements Referenceable.
func (c *StackCall) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
	return c.ResultValue(ctx, phase)
}

func (c *StackCall) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	_, moreDiags := c.CheckForEachValue(ctx, phase)
	diags = diags.Append(moreDiags)
	_, moreDiags = c.CheckInstances(ctx, phase)
	diags = diags.Append(moreDiags)

	// All of the other arguments in a stack call get evaluated separately
	// for each instance of the call, so [StackCallInstance] must deal
	// with those.

	return diags
}

// PlanChanges implements Plannable to perform "plan-time validation" of the
// stack call.
//
// This does not validate the instances of the stack call or the child stack
// instances they imply, so the plan walk driver must also call
// [StackCall.Instances] and explore the child objects directly.
func (c *StackCall) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	// Stack calls never contribute "planned changes" directly, but we
	// can potentially generate diagnostics if the call configuration is
	// invalid. This is therefore more a "plan-time validation" than actually
	// planning.
	return nil, c.checkValid(ctx, PlanPhase)
}

// References implements Referrer
func (c *StackCall) References(ctx context.Context) []stackaddrs.AbsReference {
	cfg := c.Declaration(ctx)
	var ret []stackaddrs.Reference
	ret = append(ret, ReferencesInExpr(ctx, cfg.ForEach)...)
	ret = append(ret, ReferencesInExpr(ctx, cfg.Inputs)...)
	return makeReferencesAbsolute(ret, c.Addr().Stack)
}

// RequiredComponents implements Applyable
func (c *StackCall) RequiredComponents(ctx context.Context) collections.Set[stackaddrs.AbsComponent] {
	return c.main.requiredComponentsForReferrer(ctx, c, PlanPhase)
}

// CheckApply implements Applyable.
func (c *StackCall) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	return nil, c.checkValid(ctx, ApplyPhase)
}

func (c *StackCall) tracingName() string {
	return c.Addr().String()
}

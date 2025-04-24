// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"sync"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StackCall represents a "stack" block in a stack configuration after
// its containing stacks have been expanded into stack instances.
type StackCall struct {
	addr   stackaddrs.AbsStackCall
	stack  *Stack
	config *StackCallConfig

	main *Main

	forEachValue perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
	instances    perEvalPhase[promising.Once[withDiagnostics[instancesResult[*StackCallInstance]]]]

	unknownInstancesMutex sync.Mutex
	unknownInstances      map[addrs.InstanceKey]*StackCallInstance
}

var _ Plannable = (*StackCall)(nil)
var _ Referenceable = (*StackCall)(nil)

func newStackCall(main *Main, addr stackaddrs.AbsStackCall, stack *Stack, config *StackCallConfig) *StackCall {
	return &StackCall{
		addr:             addr,
		main:             main,
		stack:            stack,
		config:           config,
		unknownInstances: make(map[addrs.InstanceKey]*StackCallInstance),
	}
}

// GetExternalRemovedBlocks fetches the removed blocks that target the stack
// instances being created by this stack call.
func (c *StackCall) GetExternalRemovedBlocks() *Removed {
	return c.stack.Removed().Next(c.addr.Item.Name)
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
		ctx, c.tracingName()+" for_each", c.forEachValue.For(phase),
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics
			cfg := c.config.config

			switch {

			case cfg.ForEach != nil:
				result, moreDiags := evaluateForEachExpr(ctx, cfg.ForEach, phase, c.stack, "stack")
				diags = diags.Append(moreDiags)
				if diags.HasErrors() {
					return cty.DynamicVal, diags
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
func (c *StackCall) Instances(ctx context.Context, phase EvalPhase) (map[addrs.InstanceKey]*StackCallInstance, bool) {
	ret, unknown, _ := c.CheckInstances(ctx, phase)
	return ret, unknown
}

func (c *StackCall) CheckInstances(ctx context.Context, phase EvalPhase) (map[addrs.InstanceKey]*StackCallInstance, bool, tfdiags.Diagnostics) {
	result, diags := doOnceWithDiags(
		ctx, c.tracingName()+" instances", c.instances.For(phase),
		func(ctx context.Context) (instancesResult[*StackCallInstance], tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics
			forEachVal, forEachValueDiags := c.CheckForEachValue(ctx, phase)

			diags = diags.Append(forEachValueDiags)
			if diags.HasErrors() {
				return instancesResult[*StackCallInstance]{}, diags
			}

			return instancesMap(forEachVal, func(ik addrs.InstanceKey, rd instances.RepetitionData) *StackCallInstance {
				return newStackCallInstance(c, ik, rd, c.stack.mode, c.stack.deferred)
			}), diags
		},
	)
	return result.insts, result.unknown, diags
}

func (c *StackCall) UnknownInstance(ctx context.Context, key addrs.InstanceKey, phase EvalPhase) *StackCallInstance {
	c.unknownInstancesMutex.Lock()
	defer c.unknownInstancesMutex.Unlock()

	if inst, ok := c.unknownInstances[key]; ok {
		return inst
	}

	forEachType := c.ForEachValue(ctx, phase).Type()
	repetitionData := instances.UnknownForEachRepetitionData(forEachType)
	if key != addrs.WildcardKey {
		repetitionData.EachKey = key.Value()
	}

	inst := newStackCallInstance(c, key, repetitionData, c.stack.mode, true)
	c.unknownInstances[key] = inst
	return inst
}

func (c *StackCall) ResultValue(ctx context.Context, phase EvalPhase) cty.Value {
	decl := c.config.config
	insts, unknown := c.Instances(ctx, phase)
	childResultType := c.config.TargetConfig().ResultType()

	switch {
	case decl.ForEach != nil:

		if unknown {
			// We don't know what instances we have, so we can't know what
			// the result will be.
			return cty.UnknownVal(cty.Map(childResultType))
		}

		if insts == nil {
			// Then we errored during instance calculation, this should have
			// already been reported.
			return cty.NilVal
		}

		// We expect that the instances all have string keys, which will
		// become the keys of a map that we're returning.
		elems := make(map[string]cty.Value, len(insts))
		for instKey, inst := range insts {
			k, ok := instKey.(addrs.StringKey)
			if !ok {
				panic(fmt.Sprintf("stack call with for_each has invalid instance key of type %T", instKey))
			}
			elems[string(k)] = inst.Stack(ctx, phase).ResultValue(ctx, phase)
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

		return inst.Stack(ctx, phase).ResultValue(ctx, phase)
	}
}

// ExprReferenceValue implements Referenceable.
func (c *StackCall) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
	return c.ResultValue(ctx, phase)
}

func (c *StackCall) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	_, _, moreDiags := c.CheckInstances(ctx, phase)
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
func (c *StackCall) References(context.Context) []stackaddrs.AbsReference {
	cfg := c.config.config
	var ret []stackaddrs.Reference
	ret = append(ret, ReferencesInExpr(cfg.ForEach)...)
	ret = append(ret, ReferencesInExpr(cfg.Inputs)...)
	ret = append(ret, referencesInTraversals(cfg.DependsOn)...)
	return makeReferencesAbsolute(ret, c.addr.Stack)
}

// CheckApply implements Applyable.
func (c *StackCall) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	return nil, c.checkValid(ctx, ApplyPhase)
}

func (c *StackCall) tracingName() string {
	return c.addr.String()
}

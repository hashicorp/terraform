// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type EphemeralComponent struct {
	addr stackaddrs.AbsEphemeralComponent

	main   *Main
	stack  *Stack
	config *EphemeralComponentConfig

	forEachValue perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
	instances    perEvalPhase[promising.Once[withDiagnostics[instancesResult[*EphemeralComponentInstance]]]]

	unknownInstancesMutex sync.Mutex
	unknownInstances      map[addrs.InstanceKey]*EphemeralComponentInstance
}

var _ Plannable = (*EphemeralComponent)(nil)
var _ Applyable = (*EphemeralComponent)(nil)
var _ Referenceable = (*EphemeralComponent)(nil)

func newEphemeralComponent(main *Main, addr stackaddrs.AbsEphemeralComponent, stack *Stack, config *EphemeralComponentConfig) *EphemeralComponent {
	return &EphemeralComponent{
		addr:             addr,
		main:             main,
		stack:            stack,
		config:           config,
		unknownInstances: make(map[addrs.InstanceKey]*EphemeralComponentInstance),
	}
}

// ForEachValue returns the result of evaluating the "for_each" expression
// for this ephemeral component, with the following exceptions:
//   - If the ephemeral component doesn't use "for_each" at all, returns [cty.NilVal].
//   - If the for_each expression is present but too invalid to evaluate,
//     returns [cty.DynamicVal] to represent that the for_each value cannot
//     be determined.
//
// A present and valid "for_each" expression produces a result that's
// guaranteed to be:
// - Either a set of strings, a map of any element type, or an object type
// - Known and not null (only the top-level value)
// - Not sensitive or ephemeral (only the top-level value)
func (c *EphemeralComponent) ForEachValue(ctx context.Context, phase EvalPhase) cty.Value {
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
// - Not sensitive or ephemeral (only the top-level value)
//
// If the diagnostics _does_ include errors then the result might be
// [cty.DynamicVal], which represents that the for_each expression was so invalid
// that we cannot know the for_each value.
func (c *EphemeralComponent) CheckForEachValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	val, diags := doOnceWithDiags(
		ctx, c.tracingName()+" for_each", c.forEachValue.For(phase),
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics
			cfg := c.config.config

			switch {

			case cfg.ForEach != nil:
				result, moreDiags := evaluateForEachExpr(ctx, cfg.ForEach, phase, c.stack, "ephemeral component")
				diags = diags.Append(moreDiags)
				if diags.HasErrors() {
					return cty.DynamicVal, diags
				}

				return result.Value, diags

			default:
				// This ephemeral component config doesn't use for_each at all
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
// will visit the ephemeral component directly and ask it to check itself, and that
// call will be the one responsible for returning any diagnostics.
func (c *EphemeralComponent) Instances(ctx context.Context, phase EvalPhase) (map[addrs.InstanceKey]*EphemeralComponentInstance, bool) {
	ret, unknown, _ := c.CheckInstances(ctx, phase)
	return ret, unknown
}

func (c *EphemeralComponent) CheckInstances(ctx context.Context, phase EvalPhase) (map[addrs.InstanceKey]*EphemeralComponentInstance, bool, tfdiags.Diagnostics) {
	result, diags := doOnceWithDiags(
		ctx, c.tracingName()+" instances", c.instances.For(phase),
		func(ctx context.Context) (instancesResult[*EphemeralComponentInstance], tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics
			forEachVal, forEachValueDiags := c.CheckForEachValue(ctx, phase)

			diags = diags.Append(forEachValueDiags)
			if diags.HasErrors() {
				return instancesResult[*EphemeralComponentInstance]{}, diags
			}

			result := instancesMap(forEachVal, func(ik addrs.InstanceKey, rd instances.RepetitionData) *EphemeralComponentInstance {
				return newEphemeralComponentInstance(c, stackaddrs.AbsEphemeralComponentToInstance(c.addr, ik), rd, c.stack.mode, c.stack.deferred)
			})

			addrs := make([]stackaddrs.AbsEphemeralComponentInstance, 0, len(result.insts))
			for _, ci := range result.insts {
				addrs = append(addrs, ci.Addr())
			}

			return result, diags
		},
	)
	return result.insts, result.unknown, diags
}

func (c *EphemeralComponent) UnknownInstance(ctx context.Context, key addrs.InstanceKey, phase EvalPhase) *EphemeralComponentInstance {
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

	inst := newEphemeralComponentInstance(c, stackaddrs.AbsEphemeralComponentToInstance(c.addr, key), repetitionData, c.stack.mode, true)
	c.unknownInstances[key] = inst
	return inst
}

func (c *EphemeralComponent) ResultValue(ctx context.Context, phase EvalPhase) cty.Value {
	val := c.resultValue(ctx, phase)
	if val == cty.NilVal {
		// Then we errored during instance calculation, this should have
		// been caught before we got here.
		return val
	}

	return val.Mark(marks.Ephemeral)
}

func (c *EphemeralComponent) resultValue(ctx context.Context, phase EvalPhase) cty.Value {
	decl := c.config.config
	insts, unknown := c.Instances(ctx, phase)

	switch {
	case decl.ForEach != nil:
		// NOTE: Unlike with StackCall, we must return object types rather than
		// map types here since the main Terraform language does not require
		// exact type constraints for its output values and so each instance of
		// a component can potentially produce a different object type.

		if unknown {
			// We can't predict the result if we don't know what the instances
			// are, so we'll return dynamic.
			return cty.DynamicVal
		}

		if insts == nil {
			// Then we errored during instance calculation, this should have
			// been caught before we got here.
			return cty.NilVal
		}

		// We expect that the instances all have string keys, which will
		// become the keys of a map that we're returning.
		elems := make(map[string]cty.Value, len(insts))
		for instKey, inst := range insts {
			k, ok := instKey.(addrs.StringKey)
			if !ok {
				panic(fmt.Sprintf("ephemeral component with for_each has invalid instance key of type %T", instKey))
			}
			elems[string(k)] = inst.ResultValue(ctx, phase)
		}
		if len(elems) == 0 {
			return cty.EmptyObjectVal
		}
		return cty.ObjectVal(elems)

	default:
		if insts == nil {
			// If we don't even know what instances we have then we can't
			// predict anything about our result.
			return cty.DynamicVal
		}
		if len(insts) != 1 {
			// Should not happen: we should have exactly one instance with addrs.NoKey
			panic("single-instance ephemeral component does not have exactly one instance")
		}
		inst, ok := insts[addrs.NoKey]
		if !ok {
			panic("single-instance ephemeral component does not have an addrs.NoKey instance")
		}
		return inst.ResultValue(ctx, phase)
	}
}

func (c *EphemeralComponent) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
	return c.ResultValue(ctx, phase)
}

func (c *EphemeralComponent) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	_, _, moreDiags := c.CheckInstances(ctx, phase)
	diags = diags.Append(moreDiags)

	return diags
}

func (c *EphemeralComponent) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	return nil, c.checkValid(ctx, PlanPhase)
}

func (c *EphemeralComponent) References(context.Context) []stackaddrs.AbsReference {
	cfg := c.config.config
	var ret []stackaddrs.Reference
	ret = append(ret, ReferencesInExpr(cfg.ForEach)...)
	ret = append(ret, ReferencesInExpr(cfg.Inputs)...)
	for _, expr := range cfg.ProviderConfigs {
		ret = append(ret, ReferencesInExpr(expr)...)
	}
	return makeReferencesAbsolute(ret, c.addr.Stack)
}

// requiredComponents returns the set of components that this ephemeral component depends on.
// This information is not stored in the plan or state, and is only meant to be used to control
// when ephemeral components can defer all provider operations internally.
func (c *EphemeralComponent) requiredComponents(ctx context.Context, phase EvalPhase) collections.Set[stackaddrs.AbsComponent] {
	return c.main.requiredComponentsForReferrer(ctx, c, phase)
}

// requiredComponentsPlanIncompleteInApply returns true if the plan being
// applied has an incomplete plan for any of the components this ephemeral
// component depends on.
//
// Ephemeral data is never stored in the plan, so ephemeral components are
// evaluated again during apply. A component with an incomplete plan doesn't
// get applied, so its output values can't be relied upon and an ephemeral
// component depending on it must not be evaluated.
func (c *EphemeralComponent) requiredComponentsPlanIncompleteInApply(ctx context.Context) bool {
	plan := c.main.PlanBeingApplied()
	for depAddr := range c.requiredComponents(ctx, ApplyPhase).All() {
		for _, depInst := range plan.ComponentInstances(depAddr) {
			if depInst.PlannedAction == plans.Delete || depInst.PlannedAction == plans.Forget {
				// A component that is being destroyed is ignored, as the output values it
				// returns come from the prior state captured in the plan, rather than from the
				// result of applying it.
				continue
			}
			if !depInst.PlanComplete {
				return true
			}
		}
	}
	return false
}

// requiredComponentsPlanIncompleteInPlan returns true if any of the required
// components of this ephemeral component has an incomplete plan, or can't be
// resolved. It can be called only during the planning phase.
func (c *EphemeralComponent) requiredComponentsPlanIncompleteInPlan(ctx context.Context) bool {
	for depAddr := range c.requiredComponents(ctx, PlanPhase).All() {
		depStack := c.main.Stack(ctx, depAddr.Stack, PlanPhase)
		if depStack == nil {
			return true // to be conservative
		}
		depComponent := depStack.Component(depAddr.Item)
		if depComponent == nil {
			return true // to be conservative
		}
		if !depComponent.PlanIsComplete(ctx) {
			return true
		}
	}
	return false
}

func (c *EphemeralComponent) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	return nil, c.checkValid(ctx, ApplyPhase)
}

func (c *EphemeralComponent) tracingName() string {
	return c.addr.String()
}

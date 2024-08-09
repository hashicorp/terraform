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
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Component struct {
	addr stackaddrs.AbsComponent

	main *Main

	forEachValue    perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
	instances       perEvalPhase[promising.Once[withDiagnostics[instancesResult[*ComponentInstance]]]]
	unknownInstance perEvalPhase[promising.Once[*ComponentInstance]]
}

var _ Plannable = (*Component)(nil)
var _ Referenceable = (*Component)(nil)

func newComponent(main *Main, addr stackaddrs.AbsComponent) *Component {
	return &Component{
		addr: addr,
		main: main,
	}
}

func (c *Component) Addr() stackaddrs.AbsComponent {
	return c.addr
}

func (c *Component) Config(ctx context.Context) *ComponentConfig {
	configAddr := stackaddrs.ConfigForAbs(c.Addr())
	stackConfig := c.main.StackConfig(ctx, configAddr.Stack)
	if stackConfig == nil {
		return nil
	}
	return stackConfig.Component(ctx, configAddr.Item)
}

func (c *Component) Declaration(ctx context.Context) *stackconfig.Component {
	cfg := c.Config(ctx)
	if cfg == nil {
		return nil
	}
	return cfg.Declaration(ctx)
}

func (c *Component) Stack(ctx context.Context) *Stack {
	// Unchecked because we should've been constructed from the same stack
	// object we're about to return, and so this should be valid unless
	// the original construction was from an invalid object itself.
	return c.main.StackUnchecked(ctx, c.Addr().Stack)
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
func (c *Component) ForEachValue(ctx context.Context, phase EvalPhase) cty.Value {
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
func (c *Component) CheckForEachValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	val, diags := doOnceWithDiags(
		ctx, c.forEachValue.For(phase), c.main,
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics
			cfg := c.Declaration(ctx)

			switch {

			case cfg.ForEach != nil:
				result, moreDiags := evaluateForEachExpr(ctx, cfg.ForEach, phase, c.Stack(ctx), "component")
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
func (c *Component) Instances(ctx context.Context, phase EvalPhase) (map[addrs.InstanceKey]*ComponentInstance, bool) {
	ret, unknown, _ := c.CheckInstances(ctx, phase)
	return ret, unknown
}

func (c *Component) CheckInstances(ctx context.Context, phase EvalPhase) (map[addrs.InstanceKey]*ComponentInstance, bool, tfdiags.Diagnostics) {
	result, diags := doOnceWithDiags(
		ctx, c.instances.For(phase), c.main,
		func(ctx context.Context) (instancesResult[*ComponentInstance], tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics
			forEachVal, forEachValueDiags := c.CheckForEachValue(ctx, phase)

			diags = diags.Append(forEachValueDiags)
			if diags.HasErrors() {
				return instancesResult[*ComponentInstance]{}, diags
			}

			result := instancesMap(forEachVal, func(ik addrs.InstanceKey, rd instances.RepetitionData) *ComponentInstance {
				return newComponentInstance(c, ik, rd)
			})

			addrs := make([]stackaddrs.AbsComponentInstance, 0, len(result.insts))
			for _, ci := range result.insts {
				addrs = append(addrs, ci.Addr())
			}

			h := hooksFromContext(ctx)
			hookSingle(ctx, h.ComponentExpanded, &hooks.ComponentInstances{
				ComponentAddr: c.Addr(),
				InstanceAddrs: addrs,
			})

			return result, diags
		},
	)
	return result.insts, result.unknown, diags
}

func (c *Component) UnknownInstance(ctx context.Context, phase EvalPhase) *ComponentInstance {
	inst, err := c.unknownInstance.For(PlanPhase).Do(ctx, func(ctx context.Context) (*ComponentInstance, error) {
		return newComponentInstance(c, addrs.WildcardKey, instances.UnknownForEachRepetitionData(c.ForEachValue(ctx, phase).Type())), nil
	})
	if err != nil {
		// Since we never return an error from the function we pass to Do,
		// this should never happen.
		panic(err)
	}
	return inst
}

func (c *Component) ResultValue(ctx context.Context, phase EvalPhase) cty.Value {
	decl := c.Declaration(ctx)
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
				panic(fmt.Sprintf("stack call with for_each has invalid instance key of type %T", instKey))
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
			panic("single-instance stack call does not have exactly one instance")
		}
		inst, ok := insts[addrs.NoKey]
		if !ok {
			panic("single-instance stack call does not have an addrs.NoKey instance")
		}
		return inst.ResultValue(ctx, phase)
	}
}

// PlanIsComplete can be called only during the planning phase, and returns
// true only if all instances of this component have "complete" plans.
//
// A component instance plan is "incomplete" if it was either created with
// resource targets set in its planning options or if the modules runtime
// decided it needed to defer at least one action for a future round.
func (c *Component) PlanIsComplete(ctx context.Context) bool {
	if !c.main.Planning() {
		panic("PlanIsComplete used when not in the planning phase")
	}
	insts, unknown := c.Instances(ctx, PlanPhase)
	if insts == nil {
		// Suggests that the configuration was not even valid enough to
		// decide what the instances are, so we'll return false to be
		// conservative and let the error be returned by a different path.
		return false
	}

	if unknown {
		// If the wildcard key is used the instance originates from an unknown
		// for_each value, which means the result is unknown.
		return false
	}

	for _, inst := range insts {
		plan := inst.ModuleTreePlan(ctx)
		if plan == nil {
			// Seems that we weren't even able to create a plan for this
			// one, so we'll just assume it was incomplete to be conservative,
			// and assume that whatever errors caused this nil result will
			// get returned by a different return path.
			return false
		}

		if !plan.Complete {
			return false
		}
	}
	// If we get here without returning false then we can say that
	// all of the instance plans are complete.
	return true
}

// ExprReferenceValue implements Referenceable.
func (c *Component) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
	return c.ResultValue(ctx, phase)
}

func (c *Component) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	_, _, moreDiags := c.CheckInstances(ctx, phase)
	diags = diags.Append(moreDiags)

	return diags
}

// PlanChanges implements Plannable by performing plan-time validation of
// the component call itself.
//
// The plan walk driver must call [Component.Instances] and also call
// PlanChanges for each instance separately in order to produce a complete
// plan.
func (c *Component) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	return nil, c.checkValid(ctx, PlanPhase)
}

// References implements Referrer
func (c *Component) References(ctx context.Context) []stackaddrs.AbsReference {
	cfg := c.Declaration(ctx)
	var ret []stackaddrs.Reference
	ret = append(ret, ReferencesInExpr(ctx, cfg.ForEach)...)
	ret = append(ret, ReferencesInExpr(ctx, cfg.Inputs)...)
	for _, expr := range cfg.ProviderConfigs {
		ret = append(ret, ReferencesInExpr(ctx, expr)...)
	}
	ret = append(ret, referencesInTraversals(ctx, cfg.DependsOn)...)
	return makeReferencesAbsolute(ret, c.Addr().Stack)
}

// RequiredComponents implements Applyable
func (c *Component) RequiredComponents(ctx context.Context) collections.Set[stackaddrs.AbsComponent] {
	return c.main.requiredComponentsForReferrer(ctx, c, PlanPhase)
}

// CheckApply implements ApplyChecker.
func (c *Component) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	return nil, c.checkValid(ctx, ApplyPhase)
}

// ApplySuccessful blocks until all instances of this component have
// completed their apply step and returns whether the apply was successful,
// or panics if called not during the apply phase.
func (c *Component) ApplySuccessful(ctx context.Context) bool {
	if !c.main.Applying() {
		panic("ApplySuccessful when not applying")
	}

	// Apply is successful if all of our instances fully completed their
	// apply phases.
	insts, _ := c.Instances(ctx, ApplyPhase)

	for _, inst := range insts {
		result := inst.ApplyResult(ctx)
		if result == nil || !result.Complete {
			return false
		}
	}

	// If we get here then either we had no instances at all or they all
	// applied completely, and so our aggregate result is success.
	return true
}

func (c *Component) tracingName() string {
	return c.Addr().String()
}

// reportNamedPromises implements namedPromiseReporter.
func (c *Component) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	name := c.Addr().String()
	instsName := name + " instances"
	forEachName := name + " for_each"
	c.instances.Each(func(ep EvalPhase, o *promising.Once[withDiagnostics[instancesResult[*ComponentInstance]]]) {
		cb(o.PromiseID(), instsName)
	})
	// FIXME: We should call reportNamedPromises on the individual
	// ComponentInstance objects too, but promising.Once doesn't allow us
	// to peek to see if the Once was already resolved without blocking on
	// it, and we don't want to block on any promises in here.
	// Without this, any promises belonging to the individual instances will
	// not be named in a self-dependency error report, but since references
	// to component instances are always indirect through the component this
	// shouldn't be a big deal in most cases.
	c.forEachValue.Each(func(ep EvalPhase, o *promising.Once[withDiagnostics[cty.Value]]) {
		cb(o.PromiseID(), forEachName)
	})
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	_ Plannable = (*Removed)(nil)
	_ Applyable = (*Removed)(nil)
)

type Removed struct {
	addr stackaddrs.AbsComponent

	main *Main

	forEachValue    perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
	instances       perEvalPhase[promising.Once[withDiagnostics[instancesResult[*RemovedInstance]]]]
	unknownInstance perEvalPhase[promising.Once[*RemovedInstance]]
}

func newRemoved(main *Main, addr stackaddrs.AbsComponent) *Removed {
	return &Removed{
		addr: addr,
		main: main,
	}
}

// reportNamedPromises implements namedPromiseReporter.
func (r *Removed) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	name := r.tracingName()
	instsName := name + " instances"
	forEachName := name + " for_each"
	r.instances.Each(func(ep EvalPhase, o *promising.Once[withDiagnostics[instancesResult[*RemovedInstance]]]) {
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
	r.forEachValue.Each(func(ep EvalPhase, o *promising.Once[withDiagnostics[cty.Value]]) {
		cb(o.PromiseID(), forEachName)
	})
}

func (r *Removed) Addr() stackaddrs.AbsComponent {
	return r.addr
}

func (r *Removed) Stack(ctx context.Context) *Stack {
	return r.main.StackUnchecked(ctx, r.addr.Stack)
}

func (r *Removed) Config(ctx context.Context) *RemovedConfig {
	configAddr := stackaddrs.ConfigForAbs(r.Addr())
	stackConfig := r.main.StackConfig(ctx, configAddr.Stack)
	if stackConfig == nil {
		return nil
	}
	return stackConfig.Removed(ctx, configAddr.Item)
}

func (r *Removed) ForEachValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return doOnceWithDiags(ctx, r.forEachValue.For(phase), r, func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
		config := r.Config(ctx).config

		switch {
		case config.ForEach != nil:
			result, diags := evaluateForEachExpr(ctx, config.ForEach, phase, r.Stack(ctx), "removed")
			if diags.HasErrors() {
				return cty.DynamicVal, diags
			}

			return result.Value, diags

		default:
			return cty.NilVal, nil
		}
	})
}

func (r *Removed) Instances(ctx context.Context, phase EvalPhase) (map[addrs.InstanceKey]*RemovedInstance, bool, tfdiags.Diagnostics) {
	result, diags := doOnceWithDiags(ctx, r.instances.For(phase), r.main, func(ctx context.Context) (instancesResult[*RemovedInstance], tfdiags.Diagnostics) {
		forEachValue, diags := r.ForEachValue(ctx, phase)
		if diags.HasErrors() {
			return instancesResult[*RemovedInstance]{}, diags
		}

		// First, evaluate the for_each value to get the set of instances the
		// user has asked to be removed.
		result := instancesMap(forEachValue, func(ik addrs.InstanceKey, rd instances.RepetitionData) *RemovedInstance {
			return newRemovedInstance(r, ik, rd, false)
		})

		// Now, filter out any instances that are not known to the previous
		// state. This means the user has targeted a component that (a) never
		// existed or (b) was removed in a previous operation.
		//
		// This stops us emitting planned and applied changes for instances that
		// do not exist.
		knownAddrs := make([]stackaddrs.AbsComponentInstance, 0, len(result.insts))
		knownInstances := make(map[addrs.InstanceKey]*RemovedInstance, len(result.insts))
		for key, ci := range result.insts {
			switch phase {
			case PlanPhase:
				if r.main.PlanPrevState().HasComponentInstance(ci.Addr()) {
					knownInstances[key] = ci
					knownAddrs = append(knownAddrs, ci.Addr())
					continue
				}
			case ApplyPhase:
				if _, ok := r.main.PlanBeingApplied().Components.GetOk(ci.Addr()); ok {
					knownInstances[key] = ci
					knownAddrs = append(knownAddrs, ci.Addr())
					continue
				}
			default:
				// Otherwise, we're running in a stage that doesn't evaluate
				// a state or the plan so we'll just include everything.
				knownInstances[key] = ci
				knownAddrs = append(knownAddrs, ci.Addr())

			}
		}
		result.insts = knownInstances

		h := hooksFromContext(ctx)
		hookSingle(ctx, h.ComponentExpanded, &hooks.ComponentInstances{
			ComponentAddr: r.Addr(),
			InstanceAddrs: knownAddrs,
		})

		return result, diags
	})
	return result.insts, result.unknown, diags
}

func (r *Removed) UnknownInstance(ctx context.Context, phase EvalPhase) *RemovedInstance {
	inst, err := r.unknownInstance.For(phase).Do(ctx, func(ctx context.Context) (*RemovedInstance, error) {
		forEachValue, _ := r.ForEachValue(ctx, phase)
		return newRemovedInstance(r, addrs.WildcardKey, instances.UnknownForEachRepetitionData(forEachValue.Type()), true), nil
	})
	if err != nil {
		panic(err)
	}
	return inst
}

func (r *Removed) PlanIsComplete(ctx context.Context) bool {
	if !r.main.Planning() {
		panic("PlanIsComplete used when not in the planning phase")
	}
	insts, unknown, _ := r.Instances(ctx, PlanPhase)
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
		plan, _ := inst.ModuleTreePlan(ctx)
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

// PlanChanges implements Plannable.
func (r *Removed) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	_, _, diags := r.Instances(ctx, PlanPhase)
	return nil, diags
}

// tracingName implements Plannable.
func (r *Removed) tracingName() string {
	return r.Addr().String() + " (removed)"
}

func (r *Removed) ApplySuccessful(ctx context.Context) bool {
	if !r.main.Applying() {
		panic("ApplySuccessful when not applying")
	}

	// Apply is successful if all of our instances fully completed their
	// apply phases.
	insts, _, _ := r.Instances(ctx, ApplyPhase)
	for _, inst := range insts {
		result, _ := inst.ApplyResult(ctx)
		if result == nil || !result.Complete {
			return false
		}
	}
	return true
}

// CheckApply implements Applyable.
func (r *Removed) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	_, _, diags := r.Instances(ctx, ApplyPhase)
	return nil, diags
}

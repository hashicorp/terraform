// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
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
	source stackaddrs.StackInstance   // the stack the removed block is defined in
	target stackaddrs.ConfigComponent // relative to source

	config *RemovedConfig
	main   *Main

	forEachValue    perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
	instances       perEvalPhase[promising.Once[withDiagnostics[instancesResult[*RemovedInstance]]]]
	unknownInstance perEvalPhase[promising.Once[*RemovedInstance]]
}

func newRemoved(main *Main, source stackaddrs.StackInstance, target stackaddrs.ConfigComponent, config *RemovedConfig) *Removed {
	return &Removed{
		source: source,
		target: target,
		main:   main,
		config: config,
	}
}

func (r *Removed) Stack(ctx context.Context) *Stack {
	return r.main.StackUnchecked(ctx, r.source)
}

func (r *Removed) Config(ctx context.Context) *RemovedConfig {
	return r.config
}

func (r *Removed) ForEachValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return doOnceWithDiags(ctx, r.tracingName()+" for_each", r.forEachValue.For(phase), func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
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

func (r *Removed) InstancesFor(ctx context.Context, target stackaddrs.StackInstance, phase EvalPhase) (map[addrs.InstanceKey]*RemovedInstance, bool, tfdiags.Diagnostics) {
	results, unknown, diags := r.Instances(ctx, phase)

	insts := make(map[addrs.InstanceKey]*RemovedInstance)
	for key, inst := range results {
		if inst.Addr().Stack.String() != target.String() {
			continue
		}
		insts[key] = inst
	}

	return insts, unknown, diags
}

func (r *Removed) Instances(ctx context.Context, phase EvalPhase) (map[addrs.InstanceKey]*RemovedInstance, bool, tfdiags.Diagnostics) {
	result, diags := doOnceWithDiags(ctx, r.tracingName()+" instances", r.instances.For(phase), func(ctx context.Context) (instancesResult[*RemovedInstance], tfdiags.Diagnostics) {
		forEachValue, diags := r.ForEachValue(ctx, phase)
		if diags.HasErrors() {
			return instancesResult[*RemovedInstance]{}, diags
		}

		// First, evaluate the for_each value to get the set of instances the
		// user has asked to be removed.
		result := instancesMap(forEachValue, func(ik addrs.InstanceKey, rd instances.RepetitionData) *RemovedInstance {
			from := r.Config(ctx).config.From

			evalContext, moreDiags := evalContextForTraversals(ctx, from.Variables(), phase, &removedInstanceExpressionScope{r, rd})
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				return nil
			}

			addr, moreDiags := from.AbsComponentInstance(evalContext, r.source)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				return nil
			}

			return newRemovedInstance(r, addr, rd, false)
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
			if ci == nil {
				// if ci is nil, then it means we couldn't process the address
				// for this instance above
				continue
			}

			// Now we know the concrete instances for this removed block,
			// we're going to verify that there are no component instances in
			// the configuration that also claim this instance.
			addr := ci.Addr()
			if stack := r.main.Stack(ctx, addr.Stack, phase); stack != nil {
				if component := stack.Component(ctx, addr.Item.Component); component != nil {
					components, _ := component.Instances(ctx, phase)
					if _, ok := components[addr.Item.Key]; ok {
						// Then this removed instance is targeting an instance
						// that is also claimed by a component block. We have to make
						// this check at this stage, because it is only now we now
						// the actual instances targeted by this removed block.
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Cannot remove component instance",
							Detail:   fmt.Sprintf("The component instance %s is targeted by a component block and cannot be removed. The relevant component is defined at %s.", addr, component.Declaration(ctx).DeclRange.ToHCL()),
							Subject:  ci.DeclRange(ctx),
						})

						// don't add this to the known instances, so only the
						// component block will return values for this instance.
						continue
					}
				}
			}

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
		hookSingle(ctx, h.RemovedExpanded, &hooks.RemovedInstances{
			Source:        r.source,
			InstanceAddrs: knownAddrs,
		})

		return result, diags
	})
	return result.insts, result.unknown, diags
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
	return fmt.Sprintf("%s -> %s (removed)", r.source, r.target)
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

var _ ExpressionScope = (*removedInstanceExpressionScope)(nil)

// removedInstanceExpressionScope is wrapper around the Removed expression
// scope that also includes repetition data for a specific instance of this
// removed block.
type removedInstanceExpressionScope struct {
	call *Removed
	rd   instances.RepetitionData
}

func (r *removedInstanceExpressionScope) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	stack := r.call.Stack(ctx)
	return stack.resolveExpressionReference(ctx, ref, nil, r.rd)
}

func (r *removedInstanceExpressionScope) PlanTimestamp() time.Time {
	return r.call.main.PlanTimestamp()
}

func (r *removedInstanceExpressionScope) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return r.call.main.ProviderFunctions(ctx, r.call.Config(ctx).StackConfig(ctx))
}

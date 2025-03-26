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
	addr stackaddrs.AbsComponent

	config *RemovedConfig
	main   *Main

	forEachValue    perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
	instances       perEvalPhase[promising.Once[withDiagnostics[instancesResult[*RemovedInstance]]]]
	unknownInstance perEvalPhase[promising.Once[*RemovedInstance]]
}

func newRemoved(main *Main, addr stackaddrs.AbsComponent, config *RemovedConfig) *Removed {
	return &Removed{
		addr:   addr,
		main:   main,
		config: config,
	}
}

func (r *Removed) Addr() stackaddrs.AbsComponent {
	return r.addr
}

func (r *Removed) Stack() *Stack {
	return r.main.StackUnchecked(r.addr.Stack)
}

func (r *Removed) Config() *RemovedConfig {
	return r.config
}

func (r *Removed) ForEachValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return doOnceWithDiags(ctx, r.tracingName()+" for_each", r.forEachValue.For(phase), func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
		config := r.Config().config

		switch {
		case config.ForEach != nil:
			result, diags := evaluateForEachExpr(ctx, config.ForEach, phase, r.Stack(), "removed")
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
	result, diags := doOnceWithDiags(ctx, r.tracingName()+" instances", r.instances.For(phase), func(ctx context.Context) (instancesResult[*RemovedInstance], tfdiags.Diagnostics) {
		forEachValue, diags := r.ForEachValue(ctx, phase)
		if diags.HasErrors() {
			return instancesResult[*RemovedInstance]{}, diags
		}

		// First, evaluate the for_each value to get the set of instances the
		// user has asked to be removed.
		result := instancesMap(forEachValue, func(ik addrs.InstanceKey, rd instances.RepetitionData) *RemovedInstance {
			expr := r.Config().config.FromIndex
			if expr == nil {
				if ik != addrs.NoKey {
					// error, but this shouldn't happen as we validate there is
					// no for each if the expression is null when parsing the
					// configuration.
					panic("has FromIndex expression, but no ForEach attribute")
				}

				from := stackaddrs.AbsComponentInstance{
					Stack: r.addr.Stack,
					Item: stackaddrs.ComponentInstance{
						Component: r.addr.Item,
						Key:       addrs.NoKey,
					},
				}

				return newRemovedInstance(r, from, rd, false)
			}

			// Otherwise, we're going to parse the FromIndex expression now.

			result, moreDiags := EvalExprAndEvalContext(ctx, expr, phase, &removedInstanceExpressionScope{r, rd})
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				return nil
			}

			key, err := addrs.ParseInstanceKey(result.Value)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity:    hcl.DiagError,
					Summary:     "Failed to parse instance key",
					Detail:      fmt.Sprintf("The `from` attribute contains an invalid instance key for the given address: %s.", err),
					Subject:     result.Expression.Range().Ptr(),
					Expression:  result.Expression,
					EvalContext: result.EvalContext,
				})
				return nil
			}

			from := stackaddrs.AbsComponentInstance{
				Stack: r.addr.Stack,
				Item: stackaddrs.ComponentInstance{
					Component: r.addr.Item,
					Key:       key,
				},
			}

			return newRemovedInstance(r, from, rd, false)
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

var _ ExpressionScope = (*removedInstanceExpressionScope)(nil)

// removedInstanceExpressionScope is wrapper around the Removed expression
// scope that also includes repetition data for a specific instance of this
// removed block.
type removedInstanceExpressionScope struct {
	call *Removed
	rd   instances.RepetitionData
}

func (r *removedInstanceExpressionScope) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	stack := r.call.Stack()
	return stack.resolveExpressionReference(ctx, ref, nil, r.rd)
}

func (r *removedInstanceExpressionScope) PlanTimestamp() time.Time {
	return r.call.main.PlanTimestamp()
}

func (r *removedInstanceExpressionScope) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return r.call.main.ProviderFunctions(ctx, r.call.Config().StackConfig())
}

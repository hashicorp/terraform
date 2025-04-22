// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var _ Plannable = (*RemovedStackCall)(nil)
var _ Applyable = (*RemovedStackCall)(nil)

type RemovedStackCall struct {
	stack  *Stack
	target stackaddrs.ConfigStackCall // relative to stack

	config *RemovedStackCallConfig

	main *Main

	forEachValue perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
	instances    perEvalPhase[promising.Once[withDiagnostics[instancesResult[*RemovedStackCallInstance]]]]

	unknownInstancesMutex sync.Mutex
	unknownInstances      collections.Map[stackaddrs.StackInstance, *RemovedStackCallInstance]
}

func newRemovedStackCall(main *Main, target stackaddrs.ConfigStackCall, stack *Stack, config *RemovedStackCallConfig) *RemovedStackCall {
	return &RemovedStackCall{
		stack:            stack,
		target:           target,
		config:           config,
		main:             main,
		unknownInstances: collections.NewMap[stackaddrs.StackInstance, *RemovedStackCallInstance](),
	}
}

// GetExternalRemovedBlocks fetches the removed blocks that target the stack
// instances being created by this stack call.
func (r *RemovedStackCall) GetExternalRemovedBlocks() *Removed {
	return r.stack.Removed().Get(r.target)
}

func (r *RemovedStackCall) ForEachValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return doOnceWithDiags(ctx, r.tracingName()+" for_each", r.forEachValue.For(phase), func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
		config := r.config.config

		switch {
		case config.ForEach != nil:
			result, diags := evaluateForEachExpr(ctx, config.ForEach, phase, r.stack, "removed")
			if diags.HasErrors() {
				return cty.DynamicVal, diags
			}

			return result.Value, diags

		default:
			return cty.NilVal, nil
		}
	})
}

func (r *RemovedStackCall) InstancesFor(ctx context.Context, stack stackaddrs.StackInstance, phase EvalPhase) (map[addrs.InstanceKey]*RemovedStackCallInstance, bool) {
	results, unknown, _ := r.Instances(ctx, phase)

	insts := make(map[addrs.InstanceKey]*RemovedStackCallInstance)
	for key, inst := range results {
		if stack.Contains(inst.from) {
			insts[key] = inst
		}
	}

	return insts, unknown
}

func (r *RemovedStackCall) Instances(ctx context.Context, phase EvalPhase) (map[addrs.InstanceKey]*RemovedStackCallInstance, bool, tfdiags.Diagnostics) {
	result, diags := doOnceWithDiags(ctx, r.tracingName()+" instances", r.instances.For(phase), func(ctx context.Context) (instancesResult[*RemovedStackCallInstance], tfdiags.Diagnostics) {
		forEachValue, diags := r.ForEachValue(ctx, phase)
		if diags.HasErrors() {
			return instancesResult[*RemovedStackCallInstance]{}, diags
		}

		// First, evaluate the for_each value to get the set of instances the
		// user has asked to be removed.
		result := instancesMap(forEachValue, func(ik addrs.InstanceKey, rd instances.RepetitionData) *RemovedStackCallInstance {
			from := r.config.config.From

			evalContext, moreDiags := evalContextForTraversals(ctx, from.Variables(), phase, &removedStackCallInstanceExpressionScope{r, rd})
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				return nil
			}

			addr, moreDiags := from.TargetStackInstance(evalContext, r.stack.addr)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				return nil
			}

			return newRemovedStackCallInstance(r, addr, rd, r.stack.deferred)
		})

		knownInstances := make(map[addrs.InstanceKey]*RemovedStackCallInstance)
		for key, rsc := range result.insts {
			if rsc == nil {
				// if rsc is nil, it means it was invalid above and we should
				// have attached diags explaining this.
				continue
			}

			if stack := r.main.Stack(ctx, rsc.from.Parent(), phase); stack != nil {
				embeddedCall := stack.EmbeddedStackCall(stackaddrs.StackCall{
					Name: rsc.from[len(rsc.from)-1].Name,
				})

				if embeddedCall != nil {
					insts, _ := embeddedCall.Instances(ctx, phase)
					if _, exists := insts[key]; exists {
						// error, we have an embedded stack call and a removed block
						// pointing at the same instance
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Cannot remove stack instance",
							Detail:   fmt.Sprintf("The stack instance %s is targeted by an embedded stack block and cannot be removed. The relevant embedded stack is defined at %s.", rsc.from, embeddedCall.config.config.DeclRange.ToHCL()),
							Subject:  rsc.call.config.config.DeclRange.ToHCL().Ptr(),
						})

						continue // don't add this to the known instances
					}
				}
			}

			switch phase {
			case PlanPhase:
				if r.main.PlanPrevState().HasStackInstance(rsc.from) {
					knownInstances[key] = rsc
					continue
				}
			case ApplyPhase:
				if stack := r.main.PlanBeingApplied().GetStack(rsc.from); stack != nil {
					knownInstances[key] = rsc
					continue
				}
			default:
				// Otherwise, we're running in a stage that doesn't evaluate
				// a state or the plan so we'll just include everything.
				knownInstances[key] = rsc
			}
		}

		result.insts = knownInstances
		return result, diags
	})
	return result.insts, result.unknown, diags
}

func (r *RemovedStackCall) UnknownInstance(ctx context.Context, from stackaddrs.StackInstance, phase EvalPhase) *RemovedStackCallInstance {
	r.unknownInstancesMutex.Lock()
	defer r.unknownInstancesMutex.Unlock()

	if inst, ok := r.unknownInstances.GetOk(from); ok {
		return inst
	}

	forEachType, _ := r.ForEachValue(ctx, phase)
	repetitionData := instances.UnknownForEachRepetitionData(forEachType.Type())

	inst := newRemovedStackCallInstance(r, from, repetitionData, true)
	r.unknownInstances.Put(from, inst)
	return inst
}

func (r *RemovedStackCall) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	_, _, diags := r.Instances(ctx, PlanPhase)
	return nil, diags
}

func (r *RemovedStackCall) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	_, _, diags := r.Instances(ctx, ApplyPhase)
	return nil, diags
}

func (r *RemovedStackCall) tracingName() string {
	return fmt.Sprintf("%s -> %s (removed)", r.stack.addr, r.target)
}

// removedStackCallInstanceExpressionScope is wrapper around the
// RemovedStackCall expression scope that also includes repetition data for a
// specific instance of this removed block.
type removedStackCallInstanceExpressionScope struct {
	call *RemovedStackCall
	rd   instances.RepetitionData
}

func (r *removedStackCallInstanceExpressionScope) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	return r.call.stack.resolveExpressionReference(ctx, ref, nil, r.rd)
}

func (r *removedStackCallInstanceExpressionScope) PlanTimestamp() time.Time {
	return r.call.main.PlanTimestamp()
}

func (r *removedStackCallInstanceExpressionScope) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return r.call.main.ProviderFunctions(ctx, r.call.stack.config)
}

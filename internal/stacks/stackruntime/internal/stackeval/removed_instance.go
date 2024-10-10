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
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	_ Plannable                                                       = (*RemovedInstance)(nil)
	_ Applyable                                                       = (*RemovedInstance)(nil)
	_ ExpressionScope                                                 = (*RemovedInstance)(nil)
	_ ConfigComponentExpressionScope[stackaddrs.AbsComponentInstance] = (*RemovedInstance)(nil)
	_ ApplyableComponentInstance                                      = (*RemovedInstance)(nil)
)

type RemovedInstance struct {
	call     *Removed
	key      addrs.InstanceKey
	deferred bool

	main *Main

	repetition instances.RepetitionData

	moduleTreePlan promising.Once[withDiagnostics[*plans.Plan]]
}

func newRemovedInstance(call *Removed, key addrs.InstanceKey, repetition instances.RepetitionData, deferred bool) *RemovedInstance {
	return &RemovedInstance{
		call:       call,
		key:        key,
		deferred:   deferred,
		main:       call.main,
		repetition: repetition,
	}
}

// reportNamedPromises implements namedPromiseReporter.
func (r *RemovedInstance) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	cb(r.moduleTreePlan.PromiseID(), r.tracingName()+" plan")
}

func (r *RemovedInstance) Addr() stackaddrs.AbsComponentInstance {
	callAddr := r.call.Addr()
	stackAddr := callAddr.Stack
	return stackaddrs.AbsComponentInstance{
		Stack: stackAddr,
		Item: stackaddrs.ComponentInstance{
			Component: callAddr.Item,
			Key:       r.key,
		},
	}
}

func (r *RemovedInstance) ModuleTreePlan(ctx context.Context) (*plans.Plan, tfdiags.Diagnostics) {
	return doOnceWithDiags(ctx, &r.moduleTreePlan, r.main, func(ctx context.Context) (*plans.Plan, tfdiags.Diagnostics) {
		var diags tfdiags.Diagnostics

		component := r.main.Stack(ctx, r.Addr().Stack, PlanPhase).Component(ctx, r.Addr().Item.Component)
		if component != nil {
			insts, unknown := component.Instances(ctx, PlanPhase)
			if !unknown {
				if _, exists := insts[r.key]; exists {
					// The instance we're planning to remove is also targeted
					// by a component block. We won't remove it, and we'll
					// report a diagnostic to that effect.
					return nil, diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Cannot remove component instance",
						Detail:   fmt.Sprintf("The component instance %s is targeted by a component block and cannot be removed. The relevant component is defined at %s.", r.Addr(), component.Declaration(ctx).DeclRange.ToHCL()),
						Subject:  r.DeclRange(ctx),
					})
				}
			}
		}

		known, unknown, moreDiags := EvalProviderValues(ctx, r.main, r.call.Config(ctx).config.ProviderConfigs, PlanPhase, r)
		if moreDiags.HasErrors() {
			// We won't actually add the diagnostics here, they should be
			// exposed via a different return path.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Cannot plan component",
				Detail:   fmt.Sprintf("Cannot generate a plan for %s because its provider configuration assignments are invalid.", r.Addr()),
				Subject:  r.DeclRange(ctx),
			})
			return nil, diags
		}

		providerClients := configuredProviderClients(ctx, r.main, known, unknown, PlanPhase)

		deferred := r.deferred
		for depAddr := range r.PlanPrevDependents(ctx).All() {
			depStack := r.main.Stack(ctx, depAddr.Stack, PlanPhase)
			if depStack == nil {
				// something weird has happened, but this means that
				// whatever thing we're depending on being deleted first
				// doesn't exist so it's fine.
				break
			}
			depComponent, depRemoved := depStack.ApplyableComponents(ctx, depAddr.Item)
			if depComponent != nil && !depComponent.PlanIsComplete(ctx) {
				deferred = true
				break
			}
			if depRemoved != nil && !depRemoved.PlanIsComplete(ctx) {
				deferred = true
				break
			}
		}

		plantimestamp := r.main.PlanTimestamp()
		forget := !r.call.Config(ctx).config.Destroy
		opts := &terraform.PlanOpts{
			Mode:                       plans.DestroyMode,
			SetVariables:               r.PlanPrevInputs(ctx),
			ExternalProviders:          providerClients,
			DeferralAllowed:            true,
			ExternalDependencyDeferred: deferred,
			Forget:                     forget,

			// We want the same plantimestamp between all components and the stacks language
			ForcePlanTimestamp: &plantimestamp,
		}

		plan, moreDiags := PlanComponentInstance(ctx, r.main, r.PlanPrevState(ctx), opts, r)
		return plan, diags.Append(moreDiags)
	})
}

// PlanPrevState returns the previous state for this component instance during
// the planning phase, or panics if called in any other phase.
func (r *RemovedInstance) PlanPrevState(ctx context.Context) *states.State {
	// The following call will panic if we aren't in the plan phase.
	stackState := r.main.PlanPrevState()
	ret := stackState.ComponentInstanceStateForModulesRuntime(r.Addr())
	if ret == nil {
		ret = states.NewState() // so caller doesn't need to worry about nil
	}
	return ret
}

// PlanPrevDependents returns the set of dependents based on the state.
func (r *RemovedInstance) PlanPrevDependents(ctx context.Context) collections.Set[stackaddrs.AbsComponent] {
	return r.main.PlanPrevState().DependentsForComponent(r.Addr())
}

func (r *RemovedInstance) PlanPrevInputs(ctx context.Context) terraform.InputValues {
	variables := r.main.PlanPrevState().InputsForComponent(r.Addr())

	inputs := make(terraform.InputValues, len(variables))
	for k, v := range variables {
		inputs[k.Name] = &terraform.InputValue{
			Value:      v,
			SourceType: terraform.ValueFromPlan,
		}
	}
	return inputs
}

func (r *RemovedInstance) PlanCurrentInputs(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	plan := r.main.PlanBeingApplied().Components.Get(r.Addr())
	inputs := make(map[string]cty.Value, len(plan.PlannedInputValues))
	for name, input := range plan.PlannedInputValues {
		value, err := input.Decode(cty.DynamicPseudoType)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Invalid input variable", fmt.Sprintf("Failed to decode the input value for %s in removed block for %s: %s", name, r.Addr(), err)))
			continue
		}

		if paths, ok := plan.PlannedInputValueMarks[name]; ok {
			inputs[name.Name] = value.MarkWithPaths(paths)
		} else {
			inputs[name.Name] = value
		}
	}
	return cty.ObjectVal(inputs), diags
}

// ApplyModuleTreePlan implements ApplyableComponentInstance.
//
// See the equivalent function within ComponentInstance for more details.
func (r *RemovedInstance) ApplyModuleTreePlan(ctx context.Context, plan *plans.Plan) (*ComponentInstanceApplyResult, tfdiags.Diagnostics) {
	if !r.main.Applying() {
		panic("called ApplyModuleTreePlan with an evaluator not instantiated for applying")
	}

	// Unlike a regular component, the removed block should have had any
	// unknown variables. With that in mind, we can just the plan directly
	// onto the shared function with no modifications.

	return ApplyComponentPlan(ctx, r.main, plan, r.call.Config(ctx).config.ProviderConfigs, r)
}

func (r *RemovedInstance) ApplyResult(ctx context.Context) (*ComponentInstanceApplyResult, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	changes := r.main.ApplyChangeResults()
	applyResult, moreDiags, err := changes.ComponentInstanceResult(ctx, r.Addr())
	diags = diags.Append(moreDiags)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Component instance apply not scheduled",
			fmt.Sprintf("Terraform needs the result from applying changes to %s, but that apply was apparently not scheduled to run: %s. This is a bug in Terraform.", r.Addr(), err),
		))
	}
	return applyResult, diags
}

func (r *RemovedInstance) PlaceholderApplyResultForSkippedApply(ctx context.Context, plan *plans.Plan) *ComponentInstanceApplyResult {
	// (We have this in here as a method just because it helps keep all of
	// the logic for constructing [ComponentInstanceApplyResult] objects
	// together in the same file, rather than having the caller synthesize
	// a result itself only in this one special situation.)
	return &ComponentInstanceApplyResult{
		FinalState: plan.PrevRunState,
		Complete:   false,
	}
}

// PlanChanges implements Plannable.
func (r *RemovedInstance) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	_, _, moreDiags := EvalProviderValues(ctx, r.main, r.call.Config(ctx).config.ProviderConfigs, PlanPhase, r)
	diags = diags.Append(moreDiags)

	plan, moreDiags := r.ModuleTreePlan(ctx)
	diags = diags.Append(moreDiags)

	var changes []stackplan.PlannedChange
	if plan != nil {
		var action plans.Action
		if r.call.Config(ctx).config.Destroy {
			action = plans.Delete
		} else {
			action = plans.Forget
		}
		changes, moreDiags = stackplan.FromPlan(ctx, r.ModuleTree(ctx), plan, nil, action, r)
		diags = diags.Append(moreDiags)
	}
	return changes, diags
}

// CheckApply implements Applyable.
func (r *RemovedInstance) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	_, _, moreDiags := EvalProviderValues(ctx, r.main, r.call.Config(ctx).config.ProviderConfigs, ApplyPhase, r)
	diags = diags.Append(moreDiags)

	inputs, moreDiags := r.PlanCurrentInputs(ctx)
	diags = diags.Append(moreDiags)

	result, moreDiags := r.ApplyResult(ctx)
	diags = diags.Append(moreDiags)

	var changes []stackstate.AppliedChange
	if result != nil {
		changes, moreDiags = stackstate.FromState(ctx, result.FinalState, r.main.PlanBeingApplied().Components.Get(r.Addr()), inputs, result.AffectedResourceInstanceObjects, r)
		diags = diags.Append(moreDiags)
	}
	return changes, diags
}

// ResolveExpressionReference implements ExpressionScope.
func (r *RemovedInstance) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	stack := r.call.Stack(ctx)
	return stack.resolveExpressionReference(ctx, ref, nil, r.repetition)
}

// PlanTimestamp implements ExpressionScope.
func (r *RemovedInstance) PlanTimestamp() time.Time {
	return r.main.PlanTimestamp()
}

// ExternalFunctions implements ExpressionScope.
func (r *RemovedInstance) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return r.main.ProviderFunctions(ctx, r.call.Config(ctx).StackConfig(ctx))
}

// ModuleTree implements ConfigComponentExpressionScope.
func (r *RemovedInstance) ModuleTree(ctx context.Context) *configs.Config {
	return r.call.Config(ctx).ModuleTree(ctx)
}

// DeclRange implements ConfigComponentExpressionScope.
func (r *RemovedInstance) DeclRange(ctx context.Context) *hcl.Range {
	return r.call.Config(ctx).config.DeclRange.ToHCL().Ptr()
}

// RequiredComponents implements stackplan.PlanProducer.
func (r *RemovedInstance) RequiredComponents(ctx context.Context) collections.Set[stackaddrs.AbsComponent] {
	// We return the dependencies from the state, based on the required
	// components when this component was last applied. In reality, destroy
	// operations require "dependents" to have been executed first but
	// we compute that in the plan phase based on the dependencies
	return r.main.PlanPrevState().DependenciesForComponent(r.Addr())
}

// ResourceSchema implements stackplan.PlanProducer.
func (r *RemovedInstance) ResourceSchema(ctx context.Context, providerTypeAddr addrs.Provider, mode addrs.ResourceMode, typ string) (*configschema.Block, error) {
	// This should not be able to fail with an error because we should
	// be retrieving the same schema that was already used to encode
	// the object we're working with. The error handling here is for
	// robustness but any error here suggests a bug in Terraform.

	providerType := r.main.ProviderType(ctx, providerTypeAddr)
	providerSchema, err := providerType.Schema(ctx)
	if err != nil {
		return nil, err
	}
	ret, _ := providerSchema.SchemaForResourceType(mode, typ)
	if ret == nil {
		return nil, fmt.Errorf("schema does not include %v %q", mode, typ)
	}
	return ret, nil
}

// tracingName implements Plannable.
func (r *RemovedInstance) tracingName() string {
	return r.Addr().String() + " (removed)"
}

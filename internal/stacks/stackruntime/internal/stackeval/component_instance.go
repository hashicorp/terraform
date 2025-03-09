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
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type ComponentInstance struct {
	call     *Component
	key      addrs.InstanceKey
	deferred bool

	main    *Main
	refresh *RefreshInstance

	repetition instances.RepetitionData

	moduleTreePlan promising.Once[withDiagnostics[*plans.Plan]]
}

var _ Applyable = (*ComponentInstance)(nil)
var _ Plannable = (*ComponentInstance)(nil)
var _ ExpressionScope = (*ComponentInstance)(nil)
var _ ConfigComponentExpressionScope[stackaddrs.AbsComponentInstance] = (*ComponentInstance)(nil)

func newComponentInstance(call *Component, key addrs.InstanceKey, repetition instances.RepetitionData, deferred bool) *ComponentInstance {
	component := &ComponentInstance{
		call:       call,
		key:        key,
		deferred:   deferred,
		main:       call.main,
		repetition: repetition,
	}
	component.refresh = newRefreshInstance(component)
	return component
}

func (c *ComponentInstance) Addr() stackaddrs.AbsComponentInstance {
	callAddr := c.call.Addr()
	stackAddr := callAddr.Stack
	return stackaddrs.AbsComponentInstance{
		Stack: stackAddr,
		Item: stackaddrs.ComponentInstance{
			Component: callAddr.Item,
			Key:       c.key,
		},
	}
}

func (c *ComponentInstance) RepetitionData() instances.RepetitionData {
	return c.repetition
}

func (c *ComponentInstance) InputVariableValues(ctx context.Context, phase EvalPhase) cty.Value {
	ret, _ := c.CheckInputVariableValues(ctx, phase)
	return ret
}

func (c *ComponentInstance) CheckInputVariableValues(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	config := c.call.Config(ctx)
	wantTy, defs := config.InputsType(ctx)
	decl := c.call.Declaration(ctx)
	varDecls := config.RootModuleVariableDecls(ctx)

	if wantTy == cty.NilType {
		// Suggests that the target module is invalid in some way, so we'll
		// just report that we don't know the input variable values and trust
		// that the module's problems will be reported by some other return
		// path.
		return cty.DynamicVal, nil
	}

	// We actually checked the errors statically already, so we only care about
	// the value here.
	val, diags := EvalComponentInputVariables(ctx, varDecls, wantTy, defs, decl, phase, c)
	if diags.HasErrors() {
		return cty.NilVal, diags
	}
	return val, diags
}

// inputValuesForModulesRuntime adapts the result of
// [ComponentInstance.InputVariableValues] to the representation that the
// main Terraform modules runtime expects.
//
// The second argument (expectedValues) is the value that the apply operation
// expects to see for the input variables, which is typically the input
// values from the plan.
//
// During the planning phase, the expectedValues should be nil, as they will
// only be checked during the apply phase.
func (c *ComponentInstance) inputValuesForModulesRuntime(ctx context.Context, phase EvalPhase) terraform.InputValues {
	valsObj := c.InputVariableValues(ctx, phase)
	if valsObj == cty.NilVal {
		return nil
	}

	// valsObj might be an unknown value during the planning phase, in which
	// case we'll return an InputValues with all of the expected variables
	// defined as unknown values of their expected type constraints. To
	// achieve that, we'll do our work with the configuration's object type
	// constraint instead of with the value we've been given directly.
	wantTy, _ := c.call.Config(ctx).InputsType(ctx)
	if wantTy == cty.NilType {
		// The configuration is too invalid for us to know what type we're
		// expecting, so we'll just bail.
		return nil
	}
	wantAttrs := wantTy.AttributeTypes()
	ret := make(terraform.InputValues, len(wantAttrs))
	for name, aty := range wantAttrs {
		v := valsObj.GetAttr(name)
		if !v.IsKnown() {
			// We'll ensure that it has the expected type even if
			// InputVariableValues didn't know what types to use.
			v = cty.UnknownVal(aty)
		}
		ret[name] = &terraform.InputValue{
			Value:      v,
			SourceType: terraform.ValueFromCaller,
		}
	}
	return ret
}

func (c *ComponentInstance) PlanOpts(ctx context.Context, mode plans.Mode, skipRefresh bool) (*terraform.PlanOpts, tfdiags.Diagnostics) {
	decl := c.call.Declaration(ctx)

	inputValues := c.inputValuesForModulesRuntime(ctx, PlanPhase)
	if inputValues == nil {
		return nil, nil
	}

	known, unknown, moreDiags := EvalProviderValues(ctx, c.main, c.call.Declaration(ctx).ProviderConfigs, PlanPhase, c)
	if moreDiags.HasErrors() {
		// We won't actually add the diagnostics here, they should be
		// exposed via a different return path.
		var diags tfdiags.Diagnostics
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Cannot plan component",
			Detail:   fmt.Sprintf("Cannot generate a plan for %s because its provider configuration assignments are invalid.", c.Addr()),
			Subject:  decl.DeclRange.ToHCL().Ptr(),
		})
	}

	providerClients := configuredProviderClients(ctx, c.main, known, unknown, PlanPhase)

	plantimestamp := c.main.PlanTimestamp()
	return &terraform.PlanOpts{
		Mode:                       mode,
		SkipRefresh:                skipRefresh,
		SetVariables:               inputValues,
		ExternalProviders:          providerClients,
		ExternalDependencyDeferred: c.deferred,
		DeferralAllowed:            true,
		// We want the same plantimestamp between all components and the stacks language
		ForcePlanTimestamp: &plantimestamp,
	}, nil
}

func (c *ComponentInstance) ModuleTreePlan(ctx context.Context) *plans.Plan {
	ret, _ := c.CheckModuleTreePlan(ctx)
	return ret
}

func (c *ComponentInstance) CheckModuleTreePlan(ctx context.Context) (*plans.Plan, tfdiags.Diagnostics) {
	if !c.main.Planning() {
		panic("called CheckModuleTreePlan with an evaluator not instantiated for planning")
	}

	return doOnceWithDiags(
		ctx, &c.moduleTreePlan, c.main,
		func(ctx context.Context) (*plans.Plan, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			mode := c.main.PlanningOpts().PlanningMode
			if mode == plans.DestroyMode {

				if !c.main.PlanPrevState().HasComponentInstance(c.Addr()) {
					// If the component instance doesn't exist in the previous
					// state at all, then we don't need to do anything.
					//
					// This means the component instance was added to the config
					// and never applied, or that it was previously destroyed
					// via an earlier destroy operation.
					//
					// Return a dummy plan:
					return &plans.Plan{
						UIMode:    plans.DestroyMode,
						Complete:  true,
						Applyable: true,
						Errored:   false,
						Timestamp: c.main.PlanTimestamp(),
						Changes:   plans.NewChangesSrc(), // no changes
					}, nil
				}

				// If we are destroying, then we are going to do the refresh
				// and destroy plan in two separate stages. This helps resolves
				// cycles within the dependency graph, as anything requiring
				// outputs from this component can read from the refresh result
				// without causing a cycle.

				refresh, moreDiags := c.refresh.Plan(ctx)
				var filteredDiags tfdiags.Diagnostics
				for _, diag := range moreDiags {
					if _, ok := addrs.DiagnosticOriginatesFromCheckRule(diag); ok && diag.Severity() == tfdiags.Warning {
						// We'll discard diagnostics from check rules here,
						// we're about to delete everything so anything not
						// valid will go away anyway.
						continue
					}
					filteredDiags = filteredDiags.Append(diag)
				}
				diags = diags.Append(filteredDiags)
				if refresh == nil {
					return nil, diags
				}

				// For the actual destroy plan, we'll skip the refresh and
				// simply use the refreshed state from the refresh plan.
				opts, moreDiags := c.PlanOpts(ctx, plans.DestroyMode, true)
				diags = diags.Append(moreDiags)
				if opts == nil {
					return nil, diags
				}

				if !refresh.Complete {
					// If the refresh was deferred, then we'll defer the destroy
					// plan as well.
					opts.ExternalDependencyDeferred = true
				} else {
					// If we're destroying this instance, then the dependencies
					// should be reversed. Unfortunately, we can't compute that
					// easily so instead we'll use the dependents computed at the
					// last apply operation.
					for depAddr := range c.PlanPrevDependents(ctx).All() {
						depStack := c.main.Stack(ctx, depAddr.Stack, PlanPhase)
						if depStack == nil {
							// something weird has happened, but this means that
							// whatever thing we're depending on being deleted first
							// doesn't exist so it's fine.
							continue
						}
						depComponent, depRemoved := depStack.ApplyableComponents(ctx, depAddr.Item)
						if depComponent != nil && !depComponent.PlanIsComplete(ctx) {
							opts.ExternalDependencyDeferred = true
							break
						}
						if depRemoved != nil && !depRemoved.PlanIsComplete(ctx) {
							opts.ExternalDependencyDeferred = true
							break
						}
					}
				}

				plan, moreDiags := PlanComponentInstance(ctx, c.main, refresh.PriorState, opts, c)
				return plan, diags.Append(moreDiags)
			}

			opts, moreDiags := c.PlanOpts(ctx, mode, false)
			diags = diags.Append(moreDiags)
			if opts == nil {
				return nil, diags
			}

			// If any of our upstream components have incomplete plans then
			// we need to force treating everything in this component as
			// deferred so we can preserve the correct dependency ordering.
			for depAddr := range c.call.RequiredComponents(ctx).All() {
				depStack := c.main.Stack(ctx, depAddr.Stack, PlanPhase)
				if depStack == nil {
					opts.ExternalDependencyDeferred = true // to be conservative
					break
				}
				depComponent := depStack.Component(ctx, depAddr.Item)
				if depComponent == nil {
					opts.ExternalDependencyDeferred = true // to be conservative
					break
				}
				if !depComponent.PlanIsComplete(ctx) {
					opts.ExternalDependencyDeferred = true
					break
				}
			}

			// The instance is also upstream deferred if the for_each value for
			// this instance or any parent stacks is unknown.
			if c.key == addrs.WildcardKey {
				opts.ExternalDependencyDeferred = true
			} else {
				for _, step := range c.call.addr.Stack {
					if step.Key == addrs.WildcardKey {
						opts.ExternalDependencyDeferred = true
						break
					}
				}
			}

			plan, moreDiags := PlanComponentInstance(ctx, c.main, c.PlanPrevState(ctx), opts, c)
			return plan, diags.Append(moreDiags)
		},
	)
}

// ApplyModuleTreePlan applies a plan returned by a previous call to
// [ComponentInstance.CheckModuleTreePlan].
//
// Applying a plan often has significant externally-visible side-effects, and
// so this method should be called only once for a given plan. In practice
// we currently ensure that is true by calling it only from the package-level
// [ApplyPlan] function, which arranges for this function to be called
// concurrently with the same method on other component instances and with
// a whole-tree walk to gather up results and diagnostics.
func (c *ComponentInstance) ApplyModuleTreePlan(ctx context.Context, plan *plans.Plan) (*ComponentInstanceApplyResult, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	if !c.main.Applying() {
		panic("called ApplyModuleTreePlan with an evaluator not instantiated for applying")
	}

	if plan.UIMode == plans.DestroyMode && plan.Changes.Empty() {
		stackPlan := c.main.PlanBeingApplied().Components.Get(c.Addr())

		// If we're destroying and there's nothing to destroy, then we can
		// consider this a no-op.
		return &ComponentInstanceApplyResult{
			FinalState:                      plan.PriorState, // after refresh
			AffectedResourceInstanceObjects: resourceInstanceObjectsAffectedByStackPlan(stackPlan),
			Complete:                        true,
		}, diags
	}

	// This is the result to return along with any errors that prevent us from
	// even starting the modules runtime apply phase. It reports that nothing
	// changed at all.
	noOpResult := c.PlaceholderApplyResultForSkippedApply(ctx, plan)

	// We'll need to make some light modifications to the plan to include
	// information we've learned in other parts of the apply walk that
	// should've filled in some unknown value placeholders. It would be rude
	// to modify the plan that our caller is holding though, so we'll
	// shallow-copy it. This is NOT a deep copy, so don't modify anything
	// that's reachable through any pointers without copying those first too.
	modifiedPlan := *plan
	inputValues := c.inputValuesForModulesRuntime(ctx, ApplyPhase)
	if inputValues == nil {
		// inputValuesForModulesRuntime uses nil (as opposed to a
		// non-nil zerolen map) to represent that the definition of
		// the input variables was so invalid that we cannot do
		// anything with it, in which case we'll just return early
		// and assume the plan walk driver will find the diagnostics
		// via another return path.
		return noOpResult, diags
	}
	// UGH: the "modules runtime"'s model of planning was designed around
	// the goal of producing a traditional Terraform CLI-style saved plan
	// file and so it has the input variable values already encoded as
	// plans.DynamicValue opaque byte arrays, and so we need to convert
	// our resolved input values into that format. It would be better
	// if plans.Plan used the typical in-memory format for input values
	// and let the plan file serializer worry about encoding, but we'll
	// defer that API change for now to avoid disrupting other codepaths.
	modifiedPlan.VariableValues = make(map[string]plans.DynamicValue, len(inputValues))
	modifiedPlan.VariableMarks = make(map[string][]cty.PathValueMarks, len(inputValues))
	for name, iv := range inputValues {
		val, pvm := iv.Value.UnmarkDeepWithPaths()
		dv, err := plans.NewDynamicValue(val, cty.DynamicPseudoType)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to encode input variable value",
				fmt.Sprintf(
					"Could not encode the value of input variable %q of %s: %s.\n\nThis is a bug in Terraform; please report it!",
					name, c.Addr(), err,
				),
			))
			continue
		}
		modifiedPlan.VariableValues[name] = dv
		modifiedPlan.VariableMarks[name] = pvm
	}
	if diags.HasErrors() {
		return noOpResult, diags
	}

	result, moreDiags := ApplyComponentPlan(ctx, c.main, &modifiedPlan, c.call.Declaration(ctx).ProviderConfigs, c)
	return result, diags.Append(moreDiags)
}

// PlanPrevState returns the previous state for this component instance during
// the planning phase, or panics if called in any other phase.
func (c *ComponentInstance) PlanPrevState(ctx context.Context) *states.State {
	// The following call will panic if we aren't in the plan phase.
	stackState := c.main.PlanPrevState()
	ret := stackState.ComponentInstanceStateForModulesRuntime(c.Addr())
	if ret == nil {
		ret = states.NewState() // so caller doesn't need to worry about nil
	}
	return ret
}

// PlanPrevDependents returns the set of dependents based on the state.
func (c *ComponentInstance) PlanPrevDependents(ctx context.Context) collections.Set[stackaddrs.AbsComponent] {
	return c.main.PlanPrevState().DependentsForComponent(c.Addr())
}

func (c *ComponentInstance) PlanPrevResult(ctx context.Context) map[addrs.OutputValue]cty.Value {
	return c.main.PlanPrevState().ResultsForComponent(c.Addr())
}

// ApplyResult returns the result from applying a plan for this object using
// [ApplyModuleTreePlan].
//
// Use the Complete field of the returned object to determine whether the
// apply ran to completion successfully enough for dependent work to proceed.
// If Complete is false then dependent work should not start, and instead
// dependents should unwind their stacks in a way that describes a no-op result.
func (c *ComponentInstance) ApplyResult(ctx context.Context) *ComponentInstanceApplyResult {
	ret, _ := c.CheckApplyResult(ctx)
	return ret
}

// CheckApplyResult returns the results from applying a plan for this object
// using [ApplyModuleTreePlan], and diagnostics describing any problems
// encountered when applying it.
func (c *ComponentInstance) CheckApplyResult(ctx context.Context) (*ComponentInstanceApplyResult, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	changes := c.main.ApplyChangeResults()
	applyResult, moreDiags, err := changes.ComponentInstanceResult(ctx, c.Addr())
	diags = diags.Append(moreDiags)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Component instance apply not scheduled",
			fmt.Sprintf("Terraform needs the result from applying changes to %s, but that apply was apparently not scheduled to run: %s. This is a bug in Terraform.", c.Addr(), err),
		))
	}
	return applyResult, diags
}

// PlaceholderApplyResultForSkippedApply returns a [ComponentInstanceApplyResult]
// which describes the hypothetical result of skipping the apply phase for
// this component instance altogether.
//
// It doesn't have any logic to check whether the apply _was_ actually skipped;
// the caller that's orchestrating the changes during the apply phase must
// decided that for itself and then choose between either calling
// [ComponentInstance.ApplyModuleTreePlan] to apply as normal, or returning
// the result of this function instead to explain that the apply was skipped.
func (c *ComponentInstance) PlaceholderApplyResultForSkippedApply(ctx context.Context, plan *plans.Plan) *ComponentInstanceApplyResult {
	// (We have this in here as a method just because it helps keep all of
	// the logic for constructing [ComponentInstanceApplyResult] objects
	// together in the same file, rather than having the caller synthesize
	// a result itself only in this one special situation.)
	return &ComponentInstanceApplyResult{
		FinalState: plan.PrevRunState,
		Complete:   false,
	}
}

// ApplyResultState returns the new state resulting from applying a plan for
// this object using [ApplyModuleTreePlan], or nil if the apply failed and
// so there is no new state to return.
func (c *ComponentInstance) ApplyResultState(ctx context.Context) *states.State {
	ret, _ := c.CheckApplyResultState(ctx)
	return ret
}

// CheckApplyResultState returns the new state resulting from applying a plan for
// this object using [ApplyModuleTreePlan] and diagnostics describing any
// problems encountered when applying it.
func (c *ComponentInstance) CheckApplyResultState(ctx context.Context) (*states.State, tfdiags.Diagnostics) {
	result, diags := c.CheckApplyResult(ctx)
	var newState *states.State
	if result != nil {
		newState = result.FinalState
	}
	return newState, diags
}

// InspectingState returns the state as captured in the snapshot provided when
// instantiating [Main] for [InspectPhase] evaluation.
func (c *ComponentInstance) InspectingState(ctx context.Context) *states.State {
	wholeState := c.main.InspectingState()
	return wholeState.ComponentInstanceStateForModulesRuntime(c.Addr())
}

func (c *ComponentInstance) ResultValue(ctx context.Context, phase EvalPhase) cty.Value {
	switch phase {
	case PlanPhase:

		if c.main.PlanningOpts().PlanningMode == plans.DestroyMode {
			// If we are running a destroy plan, then we'll return the result
			// of our refresh operation.
			return cty.ObjectVal(c.refresh.Result(ctx))
		}

		plan := c.ModuleTreePlan(ctx)
		if plan == nil {
			// Planning seems to have failed so we cannot decide a result value yet.
			// We can't do any better than DynamicVal here because in the
			// modules language output values don't have statically-declared
			// result types.
			return cty.DynamicVal
		}
		return cty.ObjectVal(stackplan.OutputsFromPlan(c.ModuleTree(ctx), plan))

	case ApplyPhase, InspectPhase:
		// As a special case, if we're applying and the planned action is
		// to destroy then we'll just return the planned output values
		// verbatim without waiting for anything, so that downstreams can
		// begin their own destroy phases before we start ours.
		if phase == ApplyPhase {
			fullPlan := c.main.PlanBeingApplied()
			ourPlan := fullPlan.Components.Get(c.Addr())
			if ourPlan == nil {
				// Weird, but we'll tolerate it.
				return cty.DynamicVal
			}

			if ourPlan.PlannedAction == plans.Delete || ourPlan.PlannedAction == plans.Forget {
				// In this case our result was already decided during the
				// planning phase, because we can't block on anything else
				// here to make sure we don't create a self-dependency
				// while our downstreams are trying to destroy themselves.
				attrs := make(map[string]cty.Value, len(ourPlan.PlannedOutputValues))
				for addr, val := range ourPlan.PlannedOutputValues {
					attrs[addr.Name] = val
				}
				return cty.ObjectVal(attrs)
			}
		}

		var state *states.State
		switch phase {
		case ApplyPhase:
			state = c.ApplyResultState(ctx)
		case InspectPhase:
			state = c.InspectingState(ctx)
		default:
			panic(fmt.Sprintf("unsupported evaluation phase %s", state)) // should not get here
		}
		if state == nil {
			// Applying seems to have failed so we cannot provide a result
			// value, and so we'll return a placeholder to help our caller
			// unwind gracefully with its own placeholder result.
			// We can't do any better than DynamicVal here because in the
			// modules language output values don't have statically-declared
			// result types.
			// (This should not typically happen in InspectPhase if the caller
			// provided a valid state snapshot, but we'll still tolerate it in
			// that case because InspectPhase is sometimes used in our unit
			// tests which might provide contrived input if testing component
			// instances is not their primary focus.)
			return cty.DynamicVal
		}

		// For apply and inspect phases we use the root module output values
		// from the state to construct our value.
		outputVals := state.RootOutputValues
		attrs := make(map[string]cty.Value, len(outputVals))
		for _, ov := range outputVals {
			name := ov.Addr.OutputValue.Name

			if ov.Sensitive {
				// For our purposes here, a static sensitive flag on the
				// output value is indistinguishable from the value having
				// been dynamically marked as sensitive.
				attrs[name] = ov.Value.Mark(marks.Sensitive)
				continue
			}

			// Otherwise, just set the value as is.
			attrs[name] = ov.Value
		}

		// If the apply operation was unsuccessful for any reason then we
		// might have some output values that are missing from the state,
		// because the state is only updated with the results of successful
		// operations. To avoid downstream errors we'll insert unknown values
		// for any declared output values that don't yet have a final value.
		//
		// The status of the apply operation will have been recorded elsewhere
		// so we don't need to worry about that here. This also ensures that
		// nothing will actually attempt to apply the unknown values here.
		config := c.call.Config(ctx).ModuleTree(ctx)
		for _, output := range config.Module.Outputs {
			if _, ok := attrs[output.Name]; !ok {
				attrs[output.Name] = cty.DynamicVal
			}
		}

		return cty.ObjectVal(attrs)

	default:
		// We can't produce a concrete value for any other phase.
		return cty.DynamicVal
	}
}

// ResolveExpressionReference implements ExpressionScope.
func (c *ComponentInstance) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	stack := c.call.Stack(ctx)
	return stack.resolveExpressionReference(ctx, ref, nil, c.repetition)
}

// ExternalFunctions implements ExpressionScope.
func (c *ComponentInstance) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return c.main.ProviderFunctions(ctx, c.call.Config(ctx).StackConfig(ctx))
}

// PlanTimestamp implements ExpressionScope, providing the timestamp at which
// the current plan is being run.
func (c *ComponentInstance) PlanTimestamp() time.Time {
	return c.main.PlanTimestamp()
}

// ModuleTree implements ConfigComponentExpressionScope.
func (c *ComponentInstance) ModuleTree(ctx context.Context) *configs.Config {
	return c.call.Config(ctx).ModuleTree(ctx)
}

// DeclRange implements ConfigComponentExpressionScope.
func (c *ComponentInstance) DeclRange(ctx context.Context) *hcl.Range {
	return c.call.Declaration(ctx).DeclRange.ToHCL().Ptr()
}

// PlanChanges implements Plannable by validating that all of the per-instance
// arguments are suitable, and then asking the main Terraform language runtime
// to produce a plan in terms of the component's selected module.
func (c *ComponentInstance) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	var changes []stackplan.PlannedChange
	var diags tfdiags.Diagnostics

	_, moreDiags := c.CheckInputVariableValues(ctx, PlanPhase)
	diags = diags.Append(moreDiags)

	_, _, moreDiags = EvalProviderValues(ctx, c.main, c.call.Declaration(ctx).ProviderConfigs, PlanPhase, c)
	diags = diags.Append(moreDiags)

	corePlan, moreDiags := c.CheckModuleTreePlan(ctx)
	diags = diags.Append(moreDiags)
	if corePlan != nil {
		existedBefore := false
		if prevState := c.main.PlanPrevState(); prevState != nil {
			existedBefore = prevState.HasComponentInstance(c.Addr())
		}
		destroying := corePlan.UIMode == plans.DestroyMode
		refreshOnly := corePlan.UIMode == plans.RefreshOnlyMode

		var action plans.Action
		switch {
		case destroying:
			action = plans.Delete
		case refreshOnly:
			action = plans.Read
		case existedBefore:
			action = plans.Update
		default:
			action = plans.Create
		}

		var refreshPlan *plans.Plan
		if c.main.PlanningOpts().PlanningMode == plans.DestroyMode {
			// if we're in destroy mode, then we did a separate refresh plan
			// so we'll make sure to pass that in as extra information the
			// FromPlan function can use.
			refreshPlan, _ = c.refresh.Plan(ctx)
		}

		changes, moreDiags = stackplan.FromPlan(ctx, c.ModuleTree(ctx), corePlan, refreshPlan, action, c)
		diags = diags.Append(moreDiags)
	}

	return changes, diags
}

// CheckApply implements Applyable.
func (c *ComponentInstance) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// FIXME: We need to report an AppliedChange object for the component
	// instance itself, and we need to emit "interim" objects representing
	// the "prior state" (refreshed) in each resource instance change in
	// the plan, so that the effect of refreshing will still get committed
	// to the state even if other downstream changes don't succeed.

	inputs, moreDiags := c.CheckInputVariableValues(ctx, ApplyPhase)
	diags = diags.Append(moreDiags)

	if inputs == cty.NilVal {
		// there was some error retrieving the input values, this should have
		// raised a diagnostic elsewhere, so we'll just use an empty object to
		// avoid panicking later.
		inputs = cty.EmptyObjectVal
	}

	_, _, moreDiags = EvalProviderValues(ctx, c.main, c.call.Declaration(ctx).ProviderConfigs, ApplyPhase, c)
	diags = diags.Append(moreDiags)

	applyResult, moreDiags := c.CheckApplyResult(ctx)
	diags = diags.Append(moreDiags)

	var changes []stackstate.AppliedChange
	if applyResult != nil {
		changes, moreDiags = stackstate.FromState(ctx, applyResult.FinalState, c.main.PlanBeingApplied().Components.Get(c.Addr()), inputs, applyResult.AffectedResourceInstanceObjects, c)
		diags = diags.Append(moreDiags)
	}
	return changes, diags
}

// ResourceSchema implements stackplan.PlanProducer.
func (c *ComponentInstance) ResourceSchema(ctx context.Context, providerTypeAddr addrs.Provider, mode addrs.ResourceMode, typ string) (*configschema.Block, error) {
	// This should not be able to fail with an error because we should
	// be retrieving the same schema that was already used to encode
	// the object we're working with. The error handling here is for
	// robustness but any error here suggests a bug in Terraform.

	providerType := c.main.ProviderType(ctx, providerTypeAddr)
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

// RequiredComponents implements stackplan.PlanProducer.
func (c *ComponentInstance) RequiredComponents(ctx context.Context) collections.Set[stackaddrs.AbsComponent] {
	return c.call.RequiredComponents(ctx)
}

func (c *ComponentInstance) tracingName() string {
	return c.Addr().String()
}

// reportNamedPromises implements namedPromiseReporter.
func (c *ComponentInstance) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	cb(c.moduleTreePlan.PromiseID(), c.Addr().String()+" plan")
}

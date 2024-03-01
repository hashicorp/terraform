// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ApplyOpts are options that affect the details of how Terraform will apply a
// previously-generated plan.
type ApplyOpts struct {
	// ExternalProviders is a set of pre-configured provider instances with
	// the same purpose as [PlanOpts.ExternalProviders].
	//
	// Callers must pass providers that are configured in a similar way as
	// the providers that were passed when creating the plan that's being
	// applied, or the results will be erratic.
	ExternalProviders map[addrs.RootProviderConfig]providers.Interface
}

// ApplyOpts creates an [ApplyOpts] with copies of all of the elements that
// are expected to propagate from plan to apply when planning and applying
// in the same process.
//
// In practice planning and applying are often separated into two different
// executions, in which case callers must retain enough information between
// plan and apply to construct an equivalent [ApplyOpts] themselves without
// using this function. This is here mainly for convenient internal use such
// as in test cases.
func (po *PlanOpts) ApplyOpts() *ApplyOpts {
	return &ApplyOpts{
		ExternalProviders: po.ExternalProviders,
	}
}

// Apply performs the actions described by the given Plan object and returns
// the resulting updated state.
//
// The given configuration *must* be the same configuration that was passed
// earlier to Context.Plan in order to create this plan.
//
// Even if the returned diagnostics contains errors, Apply always returns the
// resulting state which is likely to have been partially-updated.
//
// The [opts] argument may be nil to indicate that no special options are
// required. In that case, Apply will use a default set of options. Some
// options in [PlanOpts] when creating a plan must be echoed with equivalent
// settings during apply, so leaving opts as nil might not be valid for
// certain combinations of plan-time options.
func (c *Context) Apply(plan *plans.Plan, config *configs.Config, opts *ApplyOpts) (*states.State, tfdiags.Diagnostics) {
	state, _, diags := c.ApplyAndEval(plan, config, opts)
	return state, diags
}

// ApplyAndEval is like [Context.Apply] except that it additionally makes a
// best effort to return a [lang.Scope] which can evaluate expressions in the
// root module based on the content of the new state.
//
// The scope will be nil if the apply process doesn't complete successfully
// enough to produce a valid evaluation scope. If the returned state is nil
// then the scope will always be nil, but it's also possible for the scope
// to be nil even when the state isn't, if the apply didn't complete enough for
// the evaluation scope to produce consistent results.
func (c *Context) ApplyAndEval(plan *plans.Plan, config *configs.Config, opts *ApplyOpts) (*states.State, *lang.Scope, tfdiags.Diagnostics) {
	defer c.acquireRun("apply")()
	var diags tfdiags.Diagnostics

	if plan == nil {
		panic("cannot apply nil plan")
	}
	log.Printf("[DEBUG] Building and walking apply graph for %s plan", plan.UIMode)

	if opts == nil {
		opts = &ApplyOpts{}
	}

	if plan.Errored {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Cannot apply failed plan",
			`The given plan is incomplete due to errors during planning, and so it cannot be applied.`,
		))
		return nil, nil, diags
	}
	if !plan.Applyable {
		if plan.Changes.Empty() {
			// If a plan is not applyable but it didn't have any errors then that
			// suggests it was a "no-op" plan, which doesn't really do any
			// harm to apply, so we'll just do it but leave ourselves a note
			// in the trace log in case it ends up relevant to a bug report.
			log.Printf("[TRACE] Applying a no-op plan")
		} else {
			// This situation isn't something we expect, since our own rules
			// for what "applyable" means make this scenario impossible. We'll
			// reject it on the assumption that something very strange is
			// going on. and so better to halt than do something incorrect.
			// This error message is generic and useless because we don't
			// expect anyone to ever see it in normal use.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Cannot apply non-applyable plan",
				`The given plan is not applyable. If this seems like a bug in Terraform, then please report it!`,
			))
			return nil, nil, diags
		}
	}

	for _, rc := range plan.Changes.Resources {
		// Import is a no-op change during an apply (all the real action happens during the plan) but we'd
		// like to show some helpful output that mirrors the way we show other changes.
		if rc.Importing != nil {
			hookResourceID := HookResourceIdentity{
				Addr:         rc.Addr,
				ProviderAddr: rc.ProviderAddr.Provider,
			}
			for _, h := range c.hooks {
				// In future, we may need to call PostApplyImport separately elsewhere in the apply
				// operation. For now, though, we'll call Pre and Post hooks together.
				h.PreApplyImport(hookResourceID, *rc.Importing)
				h.PostApplyImport(hookResourceID, *rc.Importing)
			}
		}
	}

	graph, operation, moreDiags := c.applyGraph(plan, config, opts, true)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, nil, diags
	}

	moreDiags = checkExternalProviders(config, opts.ExternalProviders)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, nil, diags
	}

	workingState := plan.PriorState.DeepCopy()
	walker, walkDiags := c.walk(graph, operation, &graphWalkOpts{
		Config:                  config,
		InputState:              workingState,
		Changes:                 plan.Changes,
		Overrides:               plan.Overrides,
		ExternalProviderConfigs: opts.ExternalProviders,

		// We need to propagate the check results from the plan phase,
		// because that will tell us which checkable objects we're expecting
		// to see updated results from during the apply step.
		PlanTimeCheckResults: plan.Checks,

		// We also want to propagate the timestamp from the plan file.
		PlanTimeTimestamp: plan.Timestamp,

		ProviderFuncResults: providers.NewFunctionResultsTable(plan.ProviderFunctionResults),
	})
	diags = diags.Append(walker.NonFatalDiagnostics)
	diags = diags.Append(walkDiags)

	// After the walk is finished, we capture a simplified snapshot of the
	// check result data as part of the new state.
	walker.State.RecordCheckResults(walker.Checks)

	newState := walker.State.Close()
	if plan.UIMode == plans.DestroyMode && !diags.HasErrors() {
		// NOTE: This is a vestigial violation of the rule that we mustn't
		// use plan.UIMode to affect apply-time behavior.
		// We ideally ought to just call newState.PruneResourceHusks
		// unconditionally here, but we historically didn't and haven't yet
		// verified that it'd be safe to do so.
		newState.PruneResourceHusks()
	}

	if len(plan.TargetAddrs) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Applied changes may be incomplete",
			`The plan was created with the -target option in effect, so some changes requested in the configuration may have been ignored and the output values may not be fully updated. Run the following command to verify that no other changes are pending:
    terraform plan

Note that the -target option is not suitable for routine use, and is provided only for exceptional situations such as recovering from errors or mistakes, or when Terraform specifically suggests to use it as part of an error message.`,
		))
	}

	// FIXME: we cannot check for an empty plan for refresh-only, because root
	// outputs are always stored as changes. The final condition of the state
	// also depends on some cleanup which happens during the apply walk. It
	// would probably make more sense if applying a refresh-only plan were
	// simply just returning the planned state and checks, but some extra
	// cleanup is going to be needed to make the plan state match what apply
	// would do. For now we can copy the checks over which were overwritten
	// during the apply walk.
	// Despite the intent of UIMode, it must still be used for apply-time
	// differences in destroy plans too, so we can make use of that here as
	// well.
	if plan.UIMode == plans.RefreshOnlyMode {
		newState.CheckResults = plan.Checks.DeepCopy()
	}

	// The caller also gets access to an expression evaluation scope in the
	// root module, in case it needs to extract other information using
	// expressions, like in "terraform console" or the test harness.
	evalScope := evalScopeFromGraphWalk(walker, addrs.RootModuleInstance)

	return newState, evalScope, diags
}

func (c *Context) applyGraph(plan *plans.Plan, config *configs.Config, opts *ApplyOpts, validate bool) (*Graph, walkOperation, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	variables := InputValues{}
	for name, dyVal := range plan.VariableValues {
		val, err := dyVal.Decode(cty.DynamicPseudoType)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid variable value in plan",
				fmt.Sprintf("Invalid value for variable %q recorded in plan file: %s.", name, err),
			))
			continue
		}
		if pvm, ok := plan.VariableMarks[name]; ok {
			val = val.MarkWithPaths(pvm)
		}

		variables[name] = &InputValue{
			Value:      val,
			SourceType: ValueFromPlan,
		}
	}
	if diags.HasErrors() {
		return nil, walkApply, diags
	}

	// The plan.VariableValues field only records variables that were actually
	// set by the caller in the PlanOpts, so we may need to provide
	// placeholders for any other variables that the user didn't set, in
	// which case Terraform will once again use the default value from the
	// configuration when we visit these variables during the graph walk.
	for name := range config.Module.Variables {
		if _, ok := variables[name]; ok {
			continue
		}
		variables[name] = &InputValue{
			Value:      cty.NilVal,
			SourceType: ValueFromPlan,
		}
	}

	operation := walkApply
	if plan.UIMode == plans.DestroyMode {
		// FIXME: Due to differences in how objects must be handled in the
		// graph and evaluated during a complete destroy, we must continue to
		// use plans.DestroyMode to switch on this behavior. If all objects
		// which require special destroy handling can be tracked in the plan,
		// then this switch will no longer be needed and we can remove the
		// walkDestroy operation mode.
		// TODO: Audit that and remove walkDestroy as an operation mode.
		operation = walkDestroy
	}

	var externalProviderConfigs map[addrs.RootProviderConfig]providers.Interface
	if opts != nil {
		externalProviderConfigs = opts.ExternalProviders
	}

	graph, moreDiags := (&ApplyGraphBuilder{
		Config:                  config,
		Changes:                 plan.Changes,
		State:                   plan.PriorState,
		RootVariableValues:      variables,
		ExternalProviderConfigs: externalProviderConfigs,
		Plugins:                 c.plugins,
		Targets:                 plan.TargetAddrs,
		ForceReplace:            plan.ForceReplaceAddrs,
		Operation:               operation,
		ExternalReferences:      plan.ExternalReferences,
	}).Build(addrs.RootModuleInstance)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, walkApply, diags
	}

	return graph, operation, diags
}

// ApplyGraphForUI is a last vestage of graphs in the public interface of
// Context (as opposed to graphs as an implementation detail) intended only for
// use by the "terraform graph" command when asked to render an apply-time
// graph.
//
// The result of this is intended only for rendering ot the user as a dot
// graph, and so may change in future in order to make the result more useful
// in that context, even if drifts away from the physical graph that Terraform
// Core currently uses as an implementation detail of applying.
func (c *Context) ApplyGraphForUI(plan *plans.Plan, config *configs.Config) (*Graph, tfdiags.Diagnostics) {
	// For now though, this really is just the internal graph, confusing
	// implementation details and all.

	var diags tfdiags.Diagnostics

	graph, _, moreDiags := c.applyGraph(plan, config, nil, false)
	diags = diags.Append(moreDiags)
	return graph, diags
}

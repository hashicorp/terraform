// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/refactoring"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// PlanOpts are the various options that affect the details of how Terraform
// will build a plan.
type PlanOpts struct {
	// Mode defines what variety of plan the caller wishes to create.
	// Refer to the documentation of the plans.Mode type and its values
	// for more information.
	Mode plans.Mode

	// SkipRefresh specifies to trust that the current values for managed
	// resource instances in the prior state are accurate and to therefore
	// disable the usual step of fetching updated values for each resource
	// instance using its corresponding provider.
	SkipRefresh bool

	// PreDestroyRefresh indicated that this is being passed to a plan used to
	// refresh the state immediately before a destroy plan.
	// FIXME: This is a temporary fix to allow the pre-destroy refresh to
	// succeed. The refreshing operation during destroy must be a special case,
	// which can allow for missing instances in the state, and avoid blocking
	// on failing condition tests. The destroy plan itself should be
	// responsible for this special case of refreshing, and the separate
	// pre-destroy plan removed entirely.
	PreDestroyRefresh bool

	// SetVariables are the raw values for root module variables as provided
	// by the user who is requesting the run, prior to any normalization or
	// substitution of defaults. See the documentation for the InputValue
	// type for more information on how to correctly populate this.
	SetVariables InputValues

	// If Targets has a non-zero length then it activates targeted planning
	// mode, where Terraform will take actions only for resource instances
	// mentioned in this set and any other objects those resource instances
	// depend on.
	//
	// Targeted planning mode is intended for exceptional use only,
	// and so populating this field will cause Terraform to generate extra
	// warnings as part of the planning result.
	Targets []addrs.Targetable

	// ForceReplace is a set of resource instance addresses whose corresponding
	// objects should be forced planned for replacement if the provider's
	// plan would otherwise have been to either update the object in-place or
	// to take no action on it at all.
	//
	// A typical use of this argument is to ask Terraform to replace an object
	// which the user has determined is somehow degraded (via information from
	// outside of Terraform), thereby hopefully replacing it with a
	// fully-functional new object.
	ForceReplace []addrs.AbsResourceInstance

	// DeferralAllowed specifies that the plan is allowed to defer some actions,
	// so that a subset of the plan can be applied even if parts of it can't yet
	// be planned at all. Plans that contain deferred actions can't converge in
	// a single run, and their configuration must be planned again after the
	// dependencies of their deferred objects are in a usable state. Various
	// events can cause deferrals, including unknown values in count and
	// for_each arguments, and deferral notices from providers. If
	// DeferralAllowed is false, the plan will error upon encountering an object
	// that would be unplannable until after the apply.
	DeferralAllowed bool

	// ExternalReferences allows the external caller to pass in references to
	// nodes that should not be pruned even if they are not referenced within
	// the actual graph.
	ExternalReferences []*addrs.Reference

	// ExternalDependencyDeferred, when set, indicates that the caller
	// considers this configuration to depend on some other configuration
	// that had at least one deferred change, and therefore everything in
	// this configuration must have its changes deferred too so that the
	// overall dependency ordering would be correct.
	ExternalDependencyDeferred bool

	// Overrides provides a set of override objects that should be applied
	// during this plan.
	Overrides *mocking.Overrides

	// GenerateConfigPath tells Terraform where to write any generated
	// configuration for any ImportTargets that do not have configuration
	// already.
	//
	// If empty, then no config will be generated.
	GenerateConfigPath string

	// ExternalProviders are clients for pre-configured providers that are
	// treated as being passed into the root module from the caller. This
	// is equivalent to writing a "providers" argument inside a "module"
	// block in the Terraform language, but for the root module the caller
	// is written in Go rather than the Terraform language.
	//
	// Terraform Core will NOT call ValidateProviderConfig or ConfigureProvider
	// on any providers in this map; it's the caller's responsibility to
	// configure these providers based on information outside the scope of
	// the root module.
	ExternalProviders map[addrs.RootProviderConfig]providers.Interface

	// ForcePlanTimestamp, if not nil, will force the "plantimestamp" function
	// to return the given value instead of the time when the plan operation
	// started.
	//
	// This is here only to allow producing fixed results for tests. Don't
	// use it for main code.
	ForcePlanTimestamp *time.Time

	// Forget if set to true will cause the plan to forget all resources. This is
	// only allowd in the context of a destroy plan.
	Forget bool
}

// Plan generates an execution plan by comparing the given configuration
// with the given previous run state.
//
// The given planning options allow control of various other details of the
// planning process that are not represented directly in the configuration.
// You can use terraform.DefaultPlanOpts to generate a normal plan with no
// special options.
//
// If the returned diagnostics contains no errors then the returned plan is
// applyable, although Terraform cannot guarantee that applying it will fully
// succeed. If the returned diagnostics contains errors but this method
// still returns a non-nil Plan then the plan describes the subset of actions
// planned so far, which is not safe to apply but could potentially be used
// by the UI layer to give extra context to support understanding of the
// returned error messages.
func (c *Context) Plan(config *configs.Config, prevRunState *states.State, opts *PlanOpts) (*plans.Plan, tfdiags.Diagnostics) {
	plan, _, diags := c.PlanAndEval(config, prevRunState, opts)
	return plan, diags
}

// PlanAndEval is like [Context.Plan] except that it additionally makes a
// best effort to return a [lang.Scope] which can evaluate expressions in the
// root module based on the content of the generated plan.
//
// The scope will be nil if the planning process doesn't complete successfully
// enough to produce a valid evaluation scope. If the returned plan is nil
// then the scope will always be nil, but it's also possible for the scope
// to be nil even when the plan isn't, if the plan is not complete enough for
// the evaluation scope to produce consistent results.
func (c *Context) PlanAndEval(config *configs.Config, prevRunState *states.State, opts *PlanOpts) (*plans.Plan, *lang.Scope, tfdiags.Diagnostics) {
	defer c.acquireRun("plan")()
	var diags tfdiags.Diagnostics

	// Save the downstream functions from needing to deal with these broken situations.
	// No real callers should rely on these, but we have a bunch of old and
	// sloppy tests that don't always populate arguments properly.
	if config == nil {
		config = configs.NewEmptyConfig()
	}
	if prevRunState == nil {
		prevRunState = states.NewState()
	}
	if opts == nil {
		opts = &PlanOpts{
			Mode: plans.NormalMode,
		}
	}

	moreDiags := c.checkConfigDependencies(config)
	diags = diags.Append(moreDiags)
	moreDiags = c.checkStateDependencies(prevRunState)
	diags = diags.Append(moreDiags)

	// If required dependencies are not available then we'll bail early since
	// otherwise we're likely to just see a bunch of other errors related to
	// incompatibilities, which could be overwhelming for the user.
	if diags.HasErrors() {
		return nil, nil, diags
	}

	providerCfgDiags := checkExternalProviders(config, nil, prevRunState, opts.ExternalProviders)
	diags = diags.Append(providerCfgDiags)
	if providerCfgDiags.HasErrors() {
		return nil, nil, diags
	}

	switch opts.Mode {
	case plans.NormalMode, plans.DestroyMode:
		// OK
	case plans.RefreshOnlyMode:
		if opts.SkipRefresh {
			// The CLI layer (and other similar callers) should prevent this
			// combination of options.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Incompatible plan options",
				"Cannot skip refreshing in refresh-only mode. This is a bug in Terraform.",
			))
			return nil, nil, diags
		}
	default:
		// The CLI layer (and other similar callers) should not try to
		// create a context for a mode that Terraform Core doesn't support.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported plan mode",
			fmt.Sprintf("Terraform Core doesn't know how to handle plan mode %s. This is a bug in Terraform.", opts.Mode),
		))
		return nil, nil, diags
	}
	if len(opts.ForceReplace) > 0 && opts.Mode != plans.NormalMode {
		// The other modes don't generate no-op or update actions that we might
		// upgrade to be "replace", so doesn't make sense to combine those.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported plan mode",
			"Forcing resource instance replacement (with -replace=...) is allowed only in normal planning mode.",
		))
		return nil, nil, diags
	}
	if opts.Forget && opts.Mode != plans.DestroyMode {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported plan mode",
			"Forgetting all resources is only allowed in the context of a destroy plan. This is a bug in Terraform, please report it.",
		))
		return nil, nil, diags
	}

	// By the time we get here, we should have values defined for all of
	// the root module variables, even if some of them are "unknown". It's the
	// caller's responsibility to have already handled the decoding of these
	// from the various ways the CLI allows them to be set and to produce
	// user-friendly error messages if they are not all present, and so
	// the error message from checkInputVariables should never be seen and
	// includes language asking the user to report a bug.
	varDiags := checkInputVariables(config.Module.Variables, opts.SetVariables)
	diags = diags.Append(varDiags)

	if len(opts.Targets) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Resource targeting is in effect",
			`You are creating a plan with the -target option, which means that the result of this plan may not represent all of the changes requested by the current configuration.

The -target option is not for routine use, and is provided only for exceptional situations such as recovering from errors or mistakes, or when Terraform specifically suggests to use it as part of an error message.`,
		))
	}

	var plan *plans.Plan
	var planDiags tfdiags.Diagnostics
	var evalScope *lang.Scope
	switch opts.Mode {
	case plans.NormalMode:
		plan, evalScope, planDiags = c.plan(config, prevRunState, opts)
	case plans.DestroyMode:
		plan, evalScope, planDiags = c.destroyPlan(config, prevRunState, opts)
	case plans.RefreshOnlyMode:
		plan, evalScope, planDiags = c.refreshOnlyPlan(config, prevRunState, opts)
	default:
		panic(fmt.Sprintf("unsupported plan mode %s", opts.Mode))
	}
	diags = diags.Append(planDiags)
	// NOTE: We're intentionally not returning early when diags.HasErrors
	// here because we'll still populate other metadata below on a best-effort
	// basis to try to give the UI some extra context to return alongside the
	// error messages.

	// convert the variables into the format expected for the plan
	varVals := make(map[string]plans.DynamicValue, len(opts.SetVariables))
	varMarks := make(map[string][]cty.PathValueMarks, len(opts.SetVariables))
	applyTimeVariables := collections.NewSetCmp[string]()
	for k, iv := range opts.SetVariables {
		// If any input variables were declared as ephemeral and set to a
		// non-null value then those variables must be provided again (possibly
		// with _different_ non-null values) during the apply phase.
		if vc, ok := config.Module.Variables[k]; ok && vc.Ephemeral {
			// FIXME: We should actually do this based on the final value
			// in the named values state, rather than the value as provided
			// by the caller, so we can take into account the transforms
			// done during variable evaluation. This is a plausible starting
			// point for now, though.
			if iv.Value != cty.NilVal && !iv.Value.IsNull() {
				applyTimeVariables.Add(k)
			}
			continue
		}

		// Non-ephemeral variables must remain unchanged between plan and
		// apply, so we'll record their actual values.
		if iv.Value == cty.NilVal {
			continue // We only record values that the caller actually set
		}

		// Root variable values arriving from the traditional CLI path are
		// unmarked, as they are directly decoded from .tfvars, CLI arguments,
		// or the environment. However, variable values arriving from other
		// plans (via the coordination efforts of the stacks runtime) may have
		// gathered marks during evaluation. We must separate the value from
		// its marks here to maintain compatibility with plans.DynamicValue,
		// which cannot represent marks.
		value, pvm := iv.Value.UnmarkDeepWithPaths()

		// We use cty.DynamicPseudoType here so that we'll save both the
		// value _and_ its dynamic type in the plan, so we can recover
		// exactly the same value later.
		dv, err := plans.NewDynamicValue(value, cty.DynamicPseudoType)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to prepare variable value for plan",
				fmt.Sprintf("The value for variable %q could not be serialized to store in the plan: %s.", k, err),
			))
			continue
		}
		varVals[k] = dv
		varMarks[k] = pvm
	}

	// insert the run-specific data from the context into the plan; variables,
	// targets and provider SHAs.
	if plan != nil {
		plan.VariableValues = varVals
		plan.ApplyTimeVariables = applyTimeVariables
		if len(varMarks) > 0 {
			plan.VariableMarks = varMarks
		}
		plan.TargetAddrs = opts.Targets
	} else if !diags.HasErrors() {
		panic("nil plan but no errors")
	}

	if plan != nil {
		relevantAttrs, rDiags := c.relevantResourceAttrsForPlan(config, plan)
		diags = diags.Append(rDiags)
		plan.RelevantAttributes = relevantAttrs
	}

	if diags.HasErrors() {
		// We can't proceed further with an invalid plan, because an invalid
		// plan isn't applyable by definition.
		if plan != nil {
			// We'll explicitly mark our plan as errored so that it can't
			// be accidentally applied even though it's incomplete.
			plan.Errored = true
		}
		return plan, evalScope, diags
	}

	diags = diags.Append(c.checkApplyGraph(plan, config, opts))

	return plan, evalScope, diags
}

// checkApplyGraph builds the apply graph out of the current plan to
// check for any errors that may arise once the planned changes are added to
// the graph. This allows terraform to report errors (mostly cycles) during
// plan that would otherwise only crop up during apply
func (c *Context) checkApplyGraph(plan *plans.Plan, config *configs.Config, opts *PlanOpts) tfdiags.Diagnostics {
	if plan.Changes.Empty() {
		log.Println("[DEBUG] no planned changes, skipping apply graph check")
		return nil
	}
	log.Println("[DEBUG] building apply graph to check for errors")

	_, _, diags := c.applyGraph(plan, config, opts.ApplyOpts(), true)
	return diags
}

var DefaultPlanOpts = &PlanOpts{
	Mode: plans.NormalMode,
}

// SimplePlanOpts is a constructor to help with creating "simple" values of
// PlanOpts which only specify a mode and input variables.
//
// This helper function is primarily intended for use in straightforward
// tests that don't need any of the more "esoteric" planning options. For
// handling real user requests to run Terraform, it'd probably be better
// to construct a *PlanOpts value directly and provide a way for the user
// to set values for all of its fields.
//
// The "mode" and "setVariables" arguments become the values of the "Mode"
// and "SetVariables" fields in the result. Refer to the PlanOpts type
// documentation to learn about the meanings of those fields.
func SimplePlanOpts(mode plans.Mode, setVariables InputValues) *PlanOpts {
	return &PlanOpts{
		Mode:         mode,
		SetVariables: setVariables,
	}
}

func (c *Context) plan(config *configs.Config, prevRunState *states.State, opts *PlanOpts) (*plans.Plan, *lang.Scope, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	if opts.Mode != plans.NormalMode {
		panic(fmt.Sprintf("called Context.plan with %s", opts.Mode))
	}

	plan, evalScope, walkDiags := c.planWalk(config, prevRunState, opts)
	diags = diags.Append(walkDiags)

	return plan, evalScope, diags
}

func (c *Context) refreshOnlyPlan(config *configs.Config, prevRunState *states.State, opts *PlanOpts) (*plans.Plan, *lang.Scope, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if opts.Mode != plans.RefreshOnlyMode {
		panic(fmt.Sprintf("called Context.refreshOnlyPlan with %s", opts.Mode))
	}

	plan, evalScope, walkDiags := c.planWalk(config, prevRunState, opts)
	diags = diags.Append(walkDiags)
	if diags.HasErrors() {
		// Non-nil plan along with errors indicates a non-applyable partial
		// plan that's only suitable to be shown to the user as extra context
		// to help understand the errors.
		return plan, evalScope, diags
	}

	// If the graph builder and graph nodes correctly obeyed our directive
	// to refresh only, the set of resource changes should always be empty.
	// We'll safety-check that here so we can return a clear message about it,
	// rather than probably just generating confusing output at the UI layer.
	if len(plan.Changes.Resources) != 0 {
		// Some extra context in the logs in case the user reports this message
		// as a bug, as a starting point for debugging.
		for _, rc := range plan.Changes.Resources {
			if depKey := rc.DeposedKey; depKey == states.NotDeposed {
				log.Printf("[DEBUG] Refresh-only plan includes %s change for %s", rc.Action, rc.Addr)
			} else {
				log.Printf("[DEBUG] Refresh-only plan includes %s change for %s deposed object %s", rc.Action, rc.Addr, depKey)
			}
		}
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid refresh-only plan",
			"Terraform generated planned resource changes in a refresh-only plan. This is a bug in Terraform.",
		))
	}

	// We don't populate RelevantResources for a refresh-only plan, because
	// they never have any planned actions and so no resource can ever be
	// "relevant" per the intended meaning of that field.

	return plan, evalScope, diags
}

func (c *Context) destroyPlan(config *configs.Config, prevRunState *states.State, opts *PlanOpts) (*plans.Plan, *lang.Scope, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if opts.Mode != plans.DestroyMode {
		panic(fmt.Sprintf("called Context.destroyPlan with %s", opts.Mode))
	}

	priorState := prevRunState

	// A destroy plan starts by running Refresh to read any pending data
	// sources, and remove missing managed resources. This is required because
	// a "destroy plan" is only creating delete changes, and is essentially a
	// local operation.
	//
	// NOTE: if skipRefresh _is_ set then we'll rely on the destroy-plan walk
	// below to upgrade the prevRunState and priorState both to the latest
	// resource type schemas, so NodePlanDestroyableResourceInstance.Execute
	// must coordinate with this by taking that action only when c.skipRefresh
	// _is_ set. This coupling between the two is unfortunate but necessary
	// to work within our current structure.
	if !opts.SkipRefresh && !prevRunState.Empty() {
		log.Printf("[TRACE] Context.destroyPlan: calling Context.plan to get the effect of refreshing the prior state")
		refreshOpts := *opts
		refreshOpts.Mode = plans.NormalMode
		refreshOpts.PreDestroyRefresh = true

		// FIXME: A normal plan is required here to refresh the state, because
		// the state and configuration may not match during a destroy, and a
		// normal refresh plan can fail with evaluation errors. In the future
		// the destroy plan should take care of refreshing instances itself,
		// where the special cases of evaluation and skipping condition checks
		// can be done.
		refreshPlan, _, refreshDiags := c.plan(config, prevRunState, &refreshOpts)
		if refreshDiags.HasErrors() {
			// NOTE: Normally we'd append diagnostics regardless of whether
			// there are errors, just in case there are warnings we'd want to
			// preserve, but we're intentionally _not_ doing that here because
			// if the first plan succeeded then we'll be running another plan
			// in DestroyMode below, and we don't want to double-up any
			// warnings that both plan walks would generate.
			// (This does mean we won't show any warnings that would've been
			// unique to only this walk, but we're assuming here that if the
			// warnings aren't also applicable to a destroy plan then we'd
			// rather not show them here, because this non-destroy plan for
			// refreshing is largely an implementation detail.)
			diags = diags.Append(refreshDiags)
			return nil, nil, diags
		}

		// We'll use the refreshed state -- which is the  "prior state" from
		// the perspective of this "destroy plan" -- as the starting state
		// for our destroy-plan walk, so it can take into account if we
		// detected during refreshing that anything was already deleted outside
		// of Terraform.
		priorState = refreshPlan.PriorState.DeepCopy()

		// The refresh plan may have upgraded state for some resources, make
		// sure we store the new version.
		prevRunState = refreshPlan.PrevRunState.DeepCopy()
		log.Printf("[TRACE] Context.destroyPlan: now _really_ creating a destroy plan")
	}

	destroyPlan, evalScope, walkDiags := c.planWalk(config, priorState, opts)
	diags = diags.Append(walkDiags)
	if walkDiags.HasErrors() {
		// Non-nil plan along with errors indicates a non-applyable partial
		// plan that's only suitable to be shown to the user as extra context
		// to help understand the errors.
		return destroyPlan, evalScope, diags
	}

	if !opts.SkipRefresh {
		// If we didn't skip refreshing then we want the previous run state to
		// be the one we originally fed into the c.refreshOnlyPlan call above,
		// not the refreshed version we used for the destroy planWalk.
		destroyPlan.PrevRunState = prevRunState
	}

	relevantAttrs, rDiags := c.relevantResourceAttrsForPlan(config, destroyPlan)
	diags = diags.Append(rDiags)

	destroyPlan.RelevantAttributes = relevantAttrs
	return destroyPlan, evalScope, diags
}

func (c *Context) prePlanFindAndApplyMoves(config *configs.Config, prevRunState *states.State) ([]refactoring.MoveStatement, refactoring.MoveResults, tfdiags.Diagnostics) {
	explicitMoveStmts := refactoring.FindMoveStatements(config)
	implicitMoveStmts := refactoring.ImpliedMoveStatements(config, prevRunState, explicitMoveStmts)
	var moveStmts []refactoring.MoveStatement
	if stmtsLen := len(explicitMoveStmts) + len(implicitMoveStmts); stmtsLen > 0 {
		moveStmts = make([]refactoring.MoveStatement, 0, stmtsLen)
		moveStmts = append(moveStmts, explicitMoveStmts...)
		moveStmts = append(moveStmts, implicitMoveStmts...)
	}
	moveResults, diags := refactoring.ApplyMoves(moveStmts, prevRunState, c.plugins.ProviderFactories())
	return moveStmts, moveResults, diags
}

func (c *Context) prePlanVerifyTargetedMoves(moveResults refactoring.MoveResults, targets []addrs.Targetable) tfdiags.Diagnostics {
	if len(targets) < 1 {
		return nil // the following only matters when targeting
	}

	var diags tfdiags.Diagnostics

	var excluded []addrs.AbsResourceInstance
	for _, result := range moveResults.Changes.Values() {
		fromMatchesTarget := false
		toMatchesTarget := false
		for _, targetAddr := range targets {
			if targetAddr.TargetContains(result.From) {
				fromMatchesTarget = true
			}
			if targetAddr.TargetContains(result.To) {
				toMatchesTarget = true
			}
		}
		if !fromMatchesTarget {
			excluded = append(excluded, result.From)
		}
		if !toMatchesTarget {
			excluded = append(excluded, result.To)
		}
	}
	if len(excluded) > 0 {
		sort.Slice(excluded, func(i, j int) bool {
			return excluded[i].Less(excluded[j])
		})

		var listBuf strings.Builder
		var prevResourceAddr addrs.AbsResource
		for _, instAddr := range excluded {
			// Targeting generally ends up selecting whole resources rather
			// than individual instances, because we don't factor in
			// individual instances until DynamicExpand, so we're going to
			// always show whole resource addresses here, excluding any
			// instance keys. (This also neatly avoids dealing with the
			// different quoting styles required for string instance keys
			// on different shells, which is handy.)
			//
			// To avoid showing duplicates when we have multiple instances
			// of the same resource, we'll remember the most recent
			// resource we rendered in prevResource, which is sufficient
			// because we sorted the list of instance addresses above, and
			// our sort order always groups together instances of the same
			// resource.
			resourceAddr := instAddr.ContainingResource()
			if resourceAddr.Equal(prevResourceAddr) {
				continue
			}
			fmt.Fprintf(&listBuf, "\n  -target=%q", resourceAddr.String())
			prevResourceAddr = resourceAddr
		}
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Moved resource instances excluded by targeting",
			fmt.Sprintf(
				"Resource instances in your current state have moved to new addresses in the latest configuration. Terraform must include those resource instances while planning in order to ensure a correct result, but your -target=... options do not fully cover all of those resource instances.\n\nTo create a valid plan, either remove your -target=... options altogether or add the following additional target options:%s\n\nNote that adding these options may include further additional resource instances in your plan, in order to respect object dependencies.",
				listBuf.String(),
			),
		))
	}

	return diags
}

func (c *Context) postPlanValidateMoves(config *configs.Config, stmts []refactoring.MoveStatement, allInsts instances.Set) tfdiags.Diagnostics {
	return refactoring.ValidateMoves(stmts, config, allInsts)
}

// findImportTargets builds a list of import targets from any import blocks in
// config.
func (c *Context) findImportTargets(config *configs.Config) []*ImportTarget {
	var importTargets []*ImportTarget
	for _, ic := range config.Module.Import {
		importTargets = append(importTargets, &ImportTarget{
			Config: ic,
		})
	}
	return importTargets
}

// findForgetTargets builds a list of resources and a list of modules to be
// forgotten, based on any removed blocks in config.
func (c *Context) findForgetTargets(config *configs.Config) (forgetResources []addrs.ConfigResource, forgetModules []addrs.Module, diags tfdiags.Diagnostics) {
	removeStmts, diags := refactoring.FindRemoveStatements(config)
	if diags.HasErrors() {
		return nil, nil, diags
	}
	for _, rst := range removeStmts.Values() {
		if rst.Destroy {
			// no-op
		} else {
			if fr, ok := rst.From.(addrs.ConfigResource); ok {
				forgetResources = append(forgetResources, fr)
			} else if fm, ok := rst.From.(addrs.Module); ok {
				forgetModules = append(forgetModules, fm)
			} else {
				panic("Invalid ConfigMoveable type in remove statement")
			}
		}
	}
	return forgetResources, forgetModules, diags
}

func (c *Context) planWalk(config *configs.Config, prevRunState *states.State, opts *PlanOpts) (*plans.Plan, *lang.Scope, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	log.Printf("[DEBUG] Building and walking plan graph for %s", opts.Mode)

	prevRunState = prevRunState.DeepCopy() // don't modify the caller's object when we process the moves
	moveStmts, moveResults, moveDiags := c.prePlanFindAndApplyMoves(config, prevRunState)
	diags = diags.Append(moveDiags)
	if moveDiags.HasErrors() {
		return nil, nil, diags
	}

	// If resource targeting is in effect then it might conflict with the
	// move result.
	diags = diags.Append(c.prePlanVerifyTargetedMoves(moveResults, opts.Targets))
	if diags.HasErrors() {
		// We'll return early here, because if we have any moved resource
		// instances excluded by targeting then planning is likely to encounter
		// strange problems that may lead to confusing error messages.
		return nil, nil, diags
	}

	graph, walkOp, moreDiags := c.planGraph(config, prevRunState, opts)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return nil, nil, diags
	}

	timestamp := time.Now().UTC()
	if opts.ForcePlanTimestamp != nil {
		// Some tests use this to produce stable results to assert against.
		timestamp = *opts.ForcePlanTimestamp
	}

	var externalProviderConfigs map[addrs.RootProviderConfig]providers.Interface
	if opts != nil {
		externalProviderConfigs = opts.ExternalProviders
	}

	// If we get here then we should definitely have a non-nil "graph", which
	// we can now walk.
	changes := plans.NewChanges()

	// Initialize the results table to validate provider function calls.
	// Hold reference to this so we can store the table data in the plan file.
	providerFuncResults := providers.NewFunctionResultsTable(nil)

	walker, walkDiags := c.walk(graph, walkOp, &graphWalkOpts{
		Config:                     config,
		InputState:                 prevRunState,
		ExternalProviderConfigs:    externalProviderConfigs,
		DeferralAllowed:            opts.DeferralAllowed,
		ExternalDependencyDeferred: opts.ExternalDependencyDeferred,
		Changes:                    changes,
		MoveResults:                moveResults,
		Overrides:                  opts.Overrides,
		PlanTimeTimestamp:          timestamp,
		ProviderFuncResults:        providerFuncResults,
		Forget:                     opts.Forget,
	})
	diags = diags.Append(walker.NonFatalDiagnostics)
	diags = diags.Append(walkDiags)

	allInsts := walker.InstanceExpander.AllInstances()

	moveValidateDiags := c.postPlanValidateMoves(config, moveStmts, allInsts)
	if moveValidateDiags.HasErrors() {
		// If any of the move statements are invalid then those errors take
		// precedence over any other errors because an incomplete move graph
		// is quite likely to be the _cause_ of various errors. This oddity
		// comes from the fact that we need to apply the moves before we
		// actually validate them, because validation depends on the result
		// of first trying to plan.
		return nil, nil, moveValidateDiags
	}
	diags = diags.Append(moveValidateDiags) // might just contain warnings

	if moveResults.Blocked.Len() > 0 && !diags.HasErrors() {
		// If we had blocked moves and we're not going to be returning errors
		// then we'll report the blockers as a warning. We do this only in the
		// absense of errors because invalid move statements might well be
		// the root cause of the blockers, and so better to give an actionable
		// error message than a less-actionable warning.
		diags = diags.Append(blockedMovesWarningDiag(moveResults))
	}

	// If we reach this point with error diagnostics then "changes" is a
	// representation of the subset of changes we were able to plan before
	// we encountered errors, which we'll return as part of a non-nil plan
	// so that e.g. the UI can show what was planned so far in case that extra
	// context helps the user to understand the error messages we're returning.
	prevRunState = walker.PrevRunState.Close()

	// The refreshed state may have data resource objects which were deferred
	// to apply and cannot be serialized.
	walker.RefreshState.RemovePlannedResourceInstanceObjects()
	priorState := walker.RefreshState.Close()

	driftedResources, driftDiags := c.driftedResources(config, prevRunState, priorState, moveResults)
	diags = diags.Append(driftDiags)

	deferredResources, deferredDiags := c.deferredResources(config, walker.Deferrals.GetDeferredChanges(), priorState)
	diags = diags.Append(deferredDiags)

	var forgottenResources []string
	for _, rc := range changes.Resources {
		if rc.Action == plans.Forget {
			// TODO KEM display resource ids
			forgottenResources = append(forgottenResources, fmt.Sprintf(" - %s", rc.Addr))
		}
	}
	if len(forgottenResources) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Some objects will no longer be managed by Terraform",
			fmt.Sprintf("If you apply this plan, Terraform will discard its tracking information for the following objects, but it will not delete them:\n%s\n\nAfter applying this plan, Terraform will no longer manage these objects. You will need to import them into Terraform to manage them again.", strings.Join(forgottenResources, "\n")),
		))
	}
	schemas, schemaDiags := c.Schemas(config, prevRunState)
	// We must finish building a plan object, and cannot return early here.
	var changesSrc *plans.ChangesSrc
	var err error
	if !schemaDiags.HasErrors() {
		changesSrc, err = changes.Encode(schemas)
		if err != nil {
			diags = diags.Append(err)
		}
	}

	plan := &plans.Plan{
		UIMode:                  opts.Mode,
		Changes:                 changesSrc,
		DriftedResources:        driftedResources,
		DeferredResources:       deferredResources,
		PrevRunState:            prevRunState,
		PriorState:              priorState,
		ExternalReferences:      opts.ExternalReferences,
		Overrides:               opts.Overrides,
		Checks:                  states.NewCheckResults(walker.Checks),
		Timestamp:               timestamp,
		ProviderFunctionResults: providerFuncResults.GetHashes(),

		// Other fields get populated by Context.Plan after we return
	}

	// Our final rulings on whether the plan is "complete" and "applyable".
	// See the documentation for these plan fields to learn what exactly they
	// are intended to mean.
	if !diags.HasErrors() {
		if len(opts.Targets) == 0 && !walker.Deferrals.HaveAnyDeferrals() {
			// A plan without any targets or deferred actions should be
			// complete if we didn't encounter errors while producing it.
			log.Println("[TRACE] Plan is complete")
			plan.Complete = true
		} else {
			log.Println("[TRACE] Plan is incomplete")
		}
		if opts.Mode == plans.RefreshOnlyMode {
			// In refresh-only mode we explicitly don't expect to propose any
			// actions, but the plan is applyable if the state was changed
			// in an interesting way by the refresh step.
			plan.Applyable = !plan.PriorState.ManagedResourcesEqual(plan.PrevRunState) || !plan.PriorState.RootOutputValuesEqual(plan.PrevRunState)
		} else {
			// For other planning modes a plan is applyable if its "changes"
			// are not considered empty (by whatever rules the plans package
			// uses to decide that).
			plan.Applyable = !plan.Changes.Empty()
		}
		if plan.Applyable {
			log.Println("[TRACE] Plan is applyable")
		} else {
			log.Println("[TRACE] Plan is not applyable")
		}
	} else {
		log.Println("[WARN] Planning encountered errors, so plan is not applyable")
	}

	// The caller also gets access to an expression evaluation scope in the
	// root module, in case it needs to extract other information using
	// expressions, like in "terraform console" or the test harness.
	evalScope := evalScopeFromGraphWalk(walker, addrs.RootModuleInstance)

	return plan, evalScope, diags
}

func (c *Context) deferredResources(config *configs.Config, deferrals []*plans.DeferredResourceInstanceChange, state *states.State) ([]*plans.DeferredResourceInstanceChangeSrc, tfdiags.Diagnostics) {
	var deferredResources []*plans.DeferredResourceInstanceChangeSrc

	schemas, diags := c.Schemas(config, state)
	if diags.HasErrors() {
		return deferredResources, diags
	}

	for _, deferral := range deferrals {

		schema, _ := schemas.ResourceTypeConfig(
			deferral.Change.ProviderAddr.Provider,
			deferral.Change.Addr.Resource.Resource.Mode,
			deferral.Change.Addr.Resource.Resource.Type)

		ty := schema.ImpliedType()
		deferralSrc, err := deferral.Encode(ty)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to prepare deferred resource for plan",
				fmt.Sprintf("The deferred resource %q could not be serialized to store in the plan: %s.", deferral.Change.Addr, err)))
			continue
		}

		deferredResources = append(deferredResources, deferralSrc)
	}
	return deferredResources, diags
}

func (c *Context) planGraph(config *configs.Config, prevRunState *states.State, opts *PlanOpts) (*Graph, walkOperation, tfdiags.Diagnostics) {
	var externalProviderConfigs map[addrs.RootProviderConfig]providers.Interface
	if opts != nil {
		externalProviderConfigs = opts.ExternalProviders
	}

	switch mode := opts.Mode; mode {
	case plans.NormalMode:
		// In Normal mode we need to pay attention to import and removed blocks
		// in config so their targets can be added to the graph.
		forgetResources, forgetModules, diags := c.findForgetTargets(config)
		if diags.HasErrors() {
			return nil, walkPlan, diags
		}
		graph, diags := (&PlanGraphBuilder{
			Config:                  config,
			State:                   prevRunState,
			RootVariableValues:      opts.SetVariables,
			ExternalProviderConfigs: externalProviderConfigs,
			Plugins:                 c.plugins,
			Targets:                 opts.Targets,
			ForceReplace:            opts.ForceReplace,
			skipRefresh:             opts.SkipRefresh,
			preDestroyRefresh:       opts.PreDestroyRefresh,
			Operation:               walkPlan,
			ExternalReferences:      opts.ExternalReferences,
			Overrides:               opts.Overrides,
			ImportTargets:           c.findImportTargets(config),
			forgetResources:         forgetResources,
			forgetModules:           forgetModules,
			GenerateConfigPath:      opts.GenerateConfigPath,
			SkipGraphValidation:     c.graphOpts.SkipGraphValidation,
		}).Build(addrs.RootModuleInstance)
		return graph, walkPlan, diags
	case plans.RefreshOnlyMode:
		graph, diags := (&PlanGraphBuilder{
			Config:                  config,
			State:                   prevRunState,
			RootVariableValues:      opts.SetVariables,
			ExternalProviderConfigs: externalProviderConfigs,
			Plugins:                 c.plugins,
			Targets:                 opts.Targets,
			skipRefresh:             opts.SkipRefresh,
			skipPlanChanges:         true, // this activates "refresh only" mode.
			Operation:               walkPlan,
			ExternalReferences:      opts.ExternalReferences,
			Overrides:               opts.Overrides,
			SkipGraphValidation:     c.graphOpts.SkipGraphValidation,
		}).Build(addrs.RootModuleInstance)
		return graph, walkPlan, diags
	case plans.DestroyMode:
		graph, diags := (&PlanGraphBuilder{
			Config:                  config,
			State:                   prevRunState,
			RootVariableValues:      opts.SetVariables,
			ExternalProviderConfigs: externalProviderConfigs,
			Plugins:                 c.plugins,
			Targets:                 opts.Targets,
			skipRefresh:             opts.SkipRefresh,
			Operation:               walkPlanDestroy,
			Overrides:               opts.Overrides,
			SkipGraphValidation:     c.graphOpts.SkipGraphValidation,
		}).Build(addrs.RootModuleInstance)
		return graph, walkPlanDestroy, diags
	default:
		// The above should cover all plans.Mode values
		panic(fmt.Sprintf("unsupported plan mode %s", mode))
	}
}

// driftedResources is a best-effort attempt to compare the current and prior
// state. If we cannot decode the prior state for some reason, this should only
// return warnings to help the user correlate any missing resources in the
// report. This is known to happen when targeting a subset of resources,
// because the excluded instances will have been removed from the plan and
// not upgraded.
func (c *Context) driftedResources(config *configs.Config, oldState, newState *states.State, moves refactoring.MoveResults) ([]*plans.ResourceInstanceChangeSrc, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if newState.ManagedResourcesEqual(oldState) && moves.Changes.Len() == 0 {
		// Nothing to do, because we only detect and report drift for managed
		// resource instances.
		return nil, diags
	}

	schemas, schemaDiags := c.Schemas(config, newState)
	diags = diags.Append(schemaDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	var drs []*plans.ResourceInstanceChangeSrc

	for _, ms := range oldState.Modules {
		for _, rs := range ms.Resources {
			if rs.Addr.Resource.Mode != addrs.ManagedResourceMode {
				// Drift reporting is only for managed resources
				continue
			}

			provider := rs.ProviderConfig.Provider
			for key, oldIS := range rs.Instances {
				if oldIS.Current == nil {
					// Not interested in instances that only have deposed objects
					continue
				}
				addr := rs.Addr.Instance(key)

				// Previous run address defaults to the current address, but
				// can differ if the resource moved before refreshing
				prevRunAddr := addr
				if move, ok := moves.Changes.GetOk(addr); ok {
					prevRunAddr = move.From
				}

				newIS := newState.ResourceInstance(addr)

				schema, _ := schemas.ResourceTypeConfig(
					provider,
					addr.Resource.Resource.Mode,
					addr.Resource.Resource.Type,
				)
				if schema == nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Warning,
						"Missing resource schema from provider",
						fmt.Sprintf("No resource schema found for %s when decoding prior state", addr.Resource.Resource.Type),
					))
					continue
				}
				ty := schema.ImpliedType()

				oldObj, err := oldIS.Current.Decode(ty)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Warning,
						"Failed to decode resource from state",
						fmt.Sprintf("Error decoding %q from prior state: %s", addr.String(), err),
					))
					continue
				}

				var newObj *states.ResourceInstanceObject
				if newIS != nil && newIS.Current != nil {
					newObj, err = newIS.Current.Decode(ty)
					if err != nil {
						diags = diags.Append(tfdiags.Sourceless(
							tfdiags.Warning,
							"Failed to decode resource from state",
							fmt.Sprintf("Error decoding %q from prior state: %s", addr.String(), err),
						))
						continue
					}
				}

				var oldVal, newVal cty.Value
				oldVal = oldObj.Value
				if newObj != nil {
					newVal = newObj.Value
				} else {
					newVal = cty.NullVal(ty)
				}

				if oldVal.RawEquals(newVal) && addr.Equal(prevRunAddr) {
					// No drift if the two values are semantically equivalent
					// and no move has happened
					continue
				}

				// We can detect three types of changes after refreshing state,
				// only two of which are easily understood as "drift":
				//
				// - Resources which were deleted outside of Terraform;
				// - Resources where the object value has changed outside of
				//   Terraform;
				// - Resources which have been moved without other changes.
				//
				// All of these are returned as drift, to allow refresh-only plans
				// to present a full set of changes which will be applied.
				var action plans.Action
				switch {
				case newVal.IsNull():
					action = plans.Delete
				case !oldVal.RawEquals(newVal):
					action = plans.Update
				default:
					action = plans.NoOp
				}

				change := &plans.ResourceInstanceChange{
					Addr:         addr,
					PrevRunAddr:  prevRunAddr,
					ProviderAddr: rs.ProviderConfig,
					Change: plans.Change{
						Action: action,
						Before: oldVal,
						After:  newVal,
					},
				}

				changeSrc, err := change.Encode(ty)
				if err != nil {
					diags = diags.Append(err)
					return nil, diags
				}

				drs = append(drs, changeSrc)
			}
		}
	}

	return drs, diags
}

// PlanGraphForUI is a last vestage of graphs in the public interface of Context
// (as opposed to graphs as an implementation detail) intended only for use
// by the "terraform graph" command when asked to render a plan-time graph.
//
// The result of this is intended only for rendering to the user as a dot
// graph, and so may change in future in order to make the result more useful
// in that context, even if drifts away from the physical graph that Terraform
// Core currently uses as an implementation detail of planning.
func (c *Context) PlanGraphForUI(config *configs.Config, prevRunState *states.State, mode plans.Mode) (*Graph, tfdiags.Diagnostics) {
	// For now though, this really is just the internal graph, confusing
	// implementation details and all.

	var diags tfdiags.Diagnostics

	opts := &PlanOpts{Mode: mode}

	graph, _, moreDiags := c.planGraph(config, prevRunState, opts)
	diags = diags.Append(moreDiags)
	return graph, diags
}

func blockedMovesWarningDiag(results refactoring.MoveResults) tfdiags.Diagnostic {
	if results.Blocked.Len() < 1 {
		// Caller should check first
		panic("request to render blocked moves warning without any blocked moves")
	}

	var itemsBuf bytes.Buffer
	for _, blocked := range results.Blocked.Values() {
		fmt.Fprintf(&itemsBuf, "\n  - %s could not move to %s", blocked.Actual, blocked.Wanted)
	}

	return tfdiags.Sourceless(
		tfdiags.Warning,
		"Unresolved resource instance address changes",
		fmt.Sprintf(
			"Terraform tried to adjust resource instance addresses in the prior state based on change information recorded in the configuration, but some adjustments did not succeed due to existing objects already at the intended addresses:%s\n\nTerraform has planned to destroy these objects. If Terraform's proposed changes aren't appropriate, you must first resolve the conflicts using the \"terraform state\" subcommands and then create a new plan.",
			itemsBuf.String(),
		),
	)
}

// referenceAnalyzer returns a globalref.Analyzer object to help with
// global analysis of references within the configuration that's attached
// to the receiving context.
func (c *Context) referenceAnalyzer(config *configs.Config, state *states.State) (*globalref.Analyzer, tfdiags.Diagnostics) {
	schemas, diags := c.Schemas(config, state)
	if diags.HasErrors() {
		return nil, diags
	}
	return globalref.NewAnalyzer(config, schemas.Providers), diags
}

// relevantResourcesForPlan implements the heuristic we use to populate the
// RelevantResources field of returned plans.
func (c *Context) relevantResourceAttrsForPlan(config *configs.Config, plan *plans.Plan) ([]globalref.ResourceAttr, tfdiags.Diagnostics) {
	azr, diags := c.referenceAnalyzer(config, plan.PriorState)
	if diags.HasErrors() {
		return nil, diags
	}

	var refs []globalref.Reference
	for _, change := range plan.Changes.Resources {
		if change.Action == plans.NoOp {
			continue
		}

		moreRefs := azr.ReferencesFromResourceInstance(change.Addr)
		refs = append(refs, moreRefs...)
	}

	for _, change := range plan.Changes.Outputs {
		if change.Action == plans.NoOp {
			continue
		}

		moreRefs := azr.ReferencesFromOutputValue(change.Addr)
		refs = append(refs, moreRefs...)
	}

	var contributors []globalref.ResourceAttr

	for _, ref := range azr.ContributingResourceReferences(refs...) {
		if res, ok := ref.ResourceAttr(); ok {
			contributors = append(contributors, res)
		}
	}

	return contributors, diags
}

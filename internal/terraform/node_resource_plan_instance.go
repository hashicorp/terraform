package terraform

import (
	"fmt"
	"log"
	"sort"

	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

// NodePlannableResourceInstance represents a _single_ resource
// instance that is plannable. This means this represents a single
// count index, for example.
type NodePlannableResourceInstance struct {
	*NodeAbstractResourceInstance
	ForceCreateBeforeDestroy bool

	// skipRefresh indicates that we should skip refreshing individual instances
	skipRefresh bool

	// skipPlanChanges indicates we should skip trying to plan change actions
	// for any instances.
	skipPlanChanges bool

	// forceReplace are resource instance addresses where the user wants to
	// force generating a replace action. This set isn't pre-filtered, so
	// it might contain addresses that have nothing to do with the resource
	// that this node represents, which the node itself must therefore ignore.
	forceReplace []addrs.AbsResourceInstance

	// replaceTriggeredBy stores references from replace_triggered_by which
	// triggered this instance to be replaced.
	replaceTriggeredBy []*addrs.Reference
}

var (
	_ GraphNodeModuleInstance       = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeReferenceable        = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeReferencer           = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeConfigResource       = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeResourceInstance     = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeAttachResourceConfig = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeAttachResourceState  = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeExecutable           = (*NodePlannableResourceInstance)(nil)
)

// GraphNodeEvalable
func (n *NodePlannableResourceInstance) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	addr := n.ResourceInstanceAddr()

	// Eval info is different depending on what kind of resource this is
	switch addr.Resource.Resource.Mode {
	case addrs.ManagedResourceMode:
		return n.managedResourceExecute(ctx)
	case addrs.DataResourceMode:
		return n.dataResourceExecute(ctx)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.Config.Mode))
	}
}

func (n *NodePlannableResourceInstance) dataResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	config := n.Config
	addr := n.ResourceInstanceAddr()

	var change *plans.ResourceInstanceChange

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(validateSelfRef(addr.Resource, config.Config, providerSchema))
	if diags.HasErrors() {
		return diags
	}

	checkRuleSeverity := tfdiags.Error
	if n.skipPlanChanges {
		checkRuleSeverity = tfdiags.Warning
	}

	change, state, repeatData, planDiags := n.planDataSource(ctx, checkRuleSeverity)
	diags = diags.Append(planDiags)
	if diags.HasErrors() {
		return diags
	}

	// write the data source into both the refresh state and the
	// working state
	diags = diags.Append(n.writeResourceInstanceState(ctx, state, refreshState))
	if diags.HasErrors() {
		return diags
	}
	diags = diags.Append(n.writeResourceInstanceState(ctx, state, workingState))
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(n.writeChange(ctx, change, ""))

	// Post-conditions might block further progress. We intentionally do this
	// _after_ writing the state/diff because we want to check against
	// the result of the operation, and to fail on future operations
	// until the user makes the condition succeed.
	checkDiags := evalCheckRules(
		addrs.ResourcePostcondition,
		n.Config.Postconditions,
		ctx, addr, repeatData,
		checkRuleSeverity,
	)
	diags = diags.Append(checkDiags)

	return diags
}

func (n *NodePlannableResourceInstance) managedResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	config := n.Config
	addr := n.ResourceInstanceAddr()

	var change *plans.ResourceInstanceChange
	var instanceRefreshState *states.ResourceInstanceObject

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(validateSelfRef(addr.Resource, config.Config, providerSchema))
	if diags.HasErrors() {
		return diags
	}

	instanceRefreshState, readDiags := n.readResourceInstanceState(ctx, addr)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return diags
	}

	// We'll save a snapshot of what we just read from the state into the
	// prevRunState before we do anything else, since this will capture the
	// result of any schema upgrading that readResourceInstanceState just did,
	// but not include any out-of-band changes we might detect in in the
	// refresh step below.
	diags = diags.Append(n.writeResourceInstanceState(ctx, instanceRefreshState, prevRunState))
	if diags.HasErrors() {
		return diags
	}
	// Also the refreshState, because that should still reflect schema upgrades
	// even if it doesn't reflect upstream changes.
	diags = diags.Append(n.writeResourceInstanceState(ctx, instanceRefreshState, refreshState))
	if diags.HasErrors() {
		return diags
	}

	// In 0.13 we could be refreshing a resource with no config.
	// We should be operating on managed resource, but check here to be certain
	if n.Config == nil || n.Config.Managed == nil {
		log.Printf("[WARN] managedResourceExecute: no Managed config value found in instance state for %q", n.Addr)
	} else {
		if instanceRefreshState != nil {
			instanceRefreshState.CreateBeforeDestroy = n.Config.Managed.CreateBeforeDestroy || n.ForceCreateBeforeDestroy
		}
	}

	// Refresh, maybe
	if !n.skipRefresh {
		s, refreshDiags := n.refresh(ctx, states.NotDeposed, instanceRefreshState)
		diags = diags.Append(refreshDiags)
		if diags.HasErrors() {
			return diags
		}

		instanceRefreshState = s

		if instanceRefreshState != nil {
			// When refreshing we start by merging the stored dependencies and
			// the configured dependencies. The configured dependencies will be
			// stored to state once the changes are applied. If the plan
			// results in no changes, we will re-write these dependencies
			// below.
			instanceRefreshState.Dependencies = mergeDeps(n.Dependencies, instanceRefreshState.Dependencies)
		}

		diags = diags.Append(n.writeResourceInstanceState(ctx, instanceRefreshState, refreshState))
		if diags.HasErrors() {
			return diags
		}
	}

	// Plan the instance, unless we're in the refresh-only mode
	if !n.skipPlanChanges {

		// add this instance to n.forceReplace if replacement is triggered by
		// another change
		repData := instances.RepetitionData{}
		switch k := addr.Resource.Key.(type) {
		case addrs.IntKey:
			repData.CountIndex = k.Value()
		case addrs.StringKey:
			repData.EachKey = k.Value()
			repData.EachValue = cty.DynamicVal
		}

		diags = diags.Append(n.replaceTriggered(ctx, repData))
		if diags.HasErrors() {
			return diags
		}

		change, instancePlanState, repeatData, planDiags := n.plan(
			ctx, change, instanceRefreshState, n.ForceCreateBeforeDestroy, n.forceReplace,
		)
		diags = diags.Append(planDiags)
		if diags.HasErrors() {
			return diags
		}

		// FIXME: here we udpate the change to reflect the reason for
		// replacement, but we still overload forceReplace to get the correct
		// change planned.
		if len(n.replaceTriggeredBy) > 0 {
			change.ActionReason = plans.ResourceInstanceReplaceByTriggers
		}

		diags = diags.Append(n.checkPreventDestroy(change))
		if diags.HasErrors() {
			return diags
		}

		// FIXME: it is currently important that we write resource changes to
		// the plan (n.writeChange) before we write the corresponding state
		// (n.writeResourceInstanceState).
		//
		// This is because the planned resource state will normally have the
		// status of states.ObjectPlanned, which causes later logic to refer to
		// the contents of the plan to retrieve the resource data. Because
		// there is no shared lock between these two data structures, reversing
		// the order of these writes will cause a brief window of inconsistency
		// which can lead to a failed safety check.
		//
		// Future work should adjust these APIs such that it is impossible to
		// update these two data structures incorrectly through any objects
		// reachable via the terraform.EvalContext API.
		diags = diags.Append(n.writeChange(ctx, change, ""))

		diags = diags.Append(n.writeResourceInstanceState(ctx, instancePlanState, workingState))
		if diags.HasErrors() {
			return diags
		}

		// If this plan resulted in a NoOp, then apply won't have a chance to make
		// any changes to the stored dependencies. Since this is a NoOp we know
		// that the stored dependencies will have no effect during apply, and we can
		// write them out now.
		if change.Action == plans.NoOp && !depsEqual(instanceRefreshState.Dependencies, n.Dependencies) {
			// the refresh state will be the final state for this resource, so
			// finalize the dependencies here if they need to be updated.
			instanceRefreshState.Dependencies = n.Dependencies
			diags = diags.Append(n.writeResourceInstanceState(ctx, instanceRefreshState, refreshState))
			if diags.HasErrors() {
				return diags
			}
		}

		// Post-conditions might block completion. We intentionally do this
		// _after_ writing the state/diff because we want to check against
		// the result of the operation, and to fail on future operations
		// until the user makes the condition succeed.
		// (Note that some preconditions will end up being skipped during
		// planning, because their conditions depend on values not yet known.)
		checkDiags := evalCheckRules(
			addrs.ResourcePostcondition,
			n.Config.Postconditions,
			ctx, n.ResourceInstanceAddr(), repeatData,
			tfdiags.Error,
		)
		diags = diags.Append(checkDiags)
	} else {
		// In refresh-only mode we need to evaluate the for-each expression in
		// order to supply the value to the pre- and post-condition check
		// blocks. This has the unfortunate edge case of a refresh-only plan
		// executing with a for-each map which has the same keys but different
		// values, which could result in a post-condition check relying on that
		// value being inaccurate. Unless we decide to store the value of the
		// for-each expression in state, this is unavoidable.
		forEach, _ := evaluateForEachExpression(n.Config.ForEach, ctx)
		repeatData := EvalDataForInstanceKey(n.ResourceInstanceAddr().Resource.Key, forEach)

		checkDiags := evalCheckRules(
			addrs.ResourcePrecondition,
			n.Config.Preconditions,
			ctx, addr, repeatData,
			tfdiags.Warning,
		)
		diags = diags.Append(checkDiags)

		// Even if we don't plan changes, we do still need to at least update
		// the working state to reflect the refresh result. If not, then e.g.
		// any output values refering to this will not react to the drift.
		// (Even if we didn't actually refresh above, this will still save
		// the result of any schema upgrading we did in readResourceInstanceState.)
		diags = diags.Append(n.writeResourceInstanceState(ctx, instanceRefreshState, workingState))
		if diags.HasErrors() {
			return diags
		}

		// Here we also evaluate post-conditions after updating the working
		// state, because we want to check against the result of the refresh.
		// Unlike in normal planning mode, these checks are still evaluated
		// even if pre-conditions generated diagnostics, because we have no
		// planned changes to block.
		checkDiags = evalCheckRules(
			addrs.ResourcePostcondition,
			n.Config.Postconditions,
			ctx, addr, repeatData,
			tfdiags.Warning,
		)
		diags = diags.Append(checkDiags)
	}

	return diags
}

// replaceTriggered checks if this instance needs to be replace due to a change
// in a replace_triggered_by reference. If replacement is required, the
// instance address is added to forceReplace
func (n *NodePlannableResourceInstance) replaceTriggered(ctx EvalContext, repData instances.RepetitionData) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	for _, expr := range n.Config.TriggersReplacement {
		ref, replace, evalDiags := ctx.EvaluateReplaceTriggeredBy(expr, repData)
		diags = diags.Append(evalDiags)
		if diags.HasErrors() {
			continue
		}

		if replace {
			// FIXME: forceReplace accomplishes the same goal, however we may
			// want to communicate more information about which resource
			// triggered the replacement in the plan.
			// Rather than further complicating the plan method with more
			// options, we can refactor both of these features later.
			n.forceReplace = append(n.forceReplace, n.Addr)
			log.Printf("[DEBUG] ReplaceTriggeredBy forcing replacement of %s due to change in %s", n.Addr, ref.DisplayString())

			n.replaceTriggeredBy = append(n.replaceTriggeredBy, ref)
			break
		}
	}

	return diags
}

// mergeDeps returns the union of 2 sets of dependencies
func mergeDeps(a, b []addrs.ConfigResource) []addrs.ConfigResource {
	switch {
	case len(a) == 0:
		return b
	case len(b) == 0:
		return a
	}

	set := make(map[string]addrs.ConfigResource)

	for _, dep := range a {
		set[dep.String()] = dep
	}

	for _, dep := range b {
		set[dep.String()] = dep
	}

	newDeps := make([]addrs.ConfigResource, 0, len(set))
	for _, dep := range set {
		newDeps = append(newDeps, dep)
	}

	return newDeps
}

func depsEqual(a, b []addrs.ConfigResource) bool {
	if len(a) != len(b) {
		return false
	}

	less := func(s []addrs.ConfigResource) func(i, j int) bool {
		return func(i, j int) bool {
			return s[i].String() < s[j].String()
		}
	}

	sort.Slice(a, less(a))
	sort.Slice(b, less(b))

	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}

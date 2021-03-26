package terraform

import (
	"fmt"
	"log"
	"sort"

	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/addrs"
)

// NodePlannableResourceInstance represents a _single_ resource
// instance that is plannable. This means this represents a single
// count index, for example.
type NodePlannableResourceInstance struct {
	*NodeAbstractResourceInstance
	ForceCreateBeforeDestroy bool
	skipRefresh              bool
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
	var state *states.ResourceInstanceObject

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	state, err = n.readResourceInstanceState(ctx, addr)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(validateSelfRef(addr.Resource, config.Config, providerSchema))
	if diags.HasErrors() {
		return diags
	}

	change, state, planDiags := n.planDataSource(ctx, state)
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
	return diags
}

func (n *NodePlannableResourceInstance) managedResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	config := n.Config
	addr := n.ResourceInstanceAddr()

	var change *plans.ResourceInstanceChange
	var instanceRefreshState *states.ResourceInstanceObject
	var instancePlanState *states.ResourceInstanceObject

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(validateSelfRef(addr.Resource, config.Config, providerSchema))
	if diags.HasErrors() {
		return diags
	}

	instanceRefreshState, err = n.readResourceInstanceState(ctx, addr)
	diags = diags.Append(err)
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
		s, refreshDiags := n.refresh(ctx, instanceRefreshState)
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

	// Plan the instance
	change, instancePlanState, planDiags := n.plan(ctx, change, instanceRefreshState, n.ForceCreateBeforeDestroy)
	diags = diags.Append(planDiags)
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(n.checkPreventDestroy(change))
	if diags.HasErrors() {
		return diags
	}

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

	diags = diags.Append(n.writeChange(ctx, change, ""))
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

package terraform

import (
	"fmt"
	"log"

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

	_, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	state, err = n.readResourceInstanceState(ctx, addr)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	validateSelfRef := &EvalValidateSelfRef{
		Addr:           addr.Resource,
		Config:         config.Config,
		ProviderSchema: &providerSchema,
	}
	diags = diags.Append(validateSelfRef.Eval(ctx))
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
	diags = diags.Append(n.writeResourceInstanceState(ctx, state, nil, refreshState))
	if diags.HasErrors() {
		return diags
	}
	diags = diags.Append(n.writeResourceInstanceState(ctx, state, nil, workingState))
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

	_, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	validateSelfRef := &EvalValidateSelfRef{
		Addr:           addr.Resource,
		Config:         config.Config,
		ProviderSchema: &providerSchema,
	}
	diags = diags.Append(validateSelfRef.Eval(ctx))
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
		instanceRefreshState, refreshDiags := n.refresh(ctx, instanceRefreshState)
		diags = diags.Append(refreshDiags)
		if diags.HasErrors() {
			return diags
		}

		diags = diags.Append(n.writeResourceInstanceState(ctx, instanceRefreshState, n.Dependencies, refreshState))
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

	diags = diags.Append(n.writeResourceInstanceState(ctx, instancePlanState, n.Dependencies, workingState))
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(n.writeChange(ctx, change, ""))
	return diags
}

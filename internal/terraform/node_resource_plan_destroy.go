package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NodePlanDestroyableResourceInstance represents a resource that is ready
// to be planned for destruction.
type NodePlanDestroyableResourceInstance struct {
	*NodeAbstractResourceInstance

	// skipRefresh indicates that we should skip refreshing
	skipRefresh bool
}

var (
	_ GraphNodeModuleInstance       = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeReferenceable        = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeReferencer           = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeDestroyer            = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeConfigResource       = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeResourceInstance     = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeAttachResourceConfig = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeAttachResourceState  = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeExecutable           = (*NodePlanDestroyableResourceInstance)(nil)
	_ GraphNodeProviderConsumer     = (*NodePlanDestroyableResourceInstance)(nil)
)

// GraphNodeDestroyer
func (n *NodePlanDestroyableResourceInstance) DestroyAddr() *addrs.AbsResourceInstance {
	addr := n.ResourceInstanceAddr()
	return &addr
}

// GraphNodeEvalable
func (n *NodePlanDestroyableResourceInstance) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	addr := n.ResourceInstanceAddr()

	// Declare a bunch of variables that are used for state during
	// evaluation. These are written to by address in the EvalNodes we
	// declare below.
	var change *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject

	state, err := n.readResourceInstanceState(ctx, addr)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	// If we are in the "skip refresh" mode then we will have skipped over our
	// usual opportunity to update the previous run state and refresh state
	// with the result of any provider schema upgrades, so we'll compensate
	// by doing that here.
	//
	// NOTE: this is coupled with logic in Context.destroyPlan which skips
	// running a normal plan walk when refresh is enabled. These two
	// conditionals must agree (be exactly opposite) in order to get the
	// correct behavior in both cases.
	if n.skipRefresh {
		diags = diags.Append(n.writeResourceInstanceState(ctx, state, prevRunState))
		if diags.HasErrors() {
			return diags
		}
		diags = diags.Append(n.writeResourceInstanceState(ctx, state, refreshState))
		if diags.HasErrors() {
			return diags
		}
	}

	change, destroyPlanDiags := n.planDestroy(ctx, state, "")
	diags = diags.Append(destroyPlanDiags)
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(n.checkPreventDestroy(change))
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(n.writeChange(ctx, change, ""))
	return diags
}

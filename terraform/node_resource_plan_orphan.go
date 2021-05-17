package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NodePlannableResourceInstanceOrphan represents a resource that is "applyable":
// it is ready to be applied and is represented by a diff.
type NodePlannableResourceInstanceOrphan struct {
	*NodeAbstractResourceInstance

	// skipRefresh indicates that we should skip refreshing individual instances
	skipRefresh bool

	// skipPlanChanges indicates we should skip trying to plan change actions
	// for any instances.
	skipPlanChanges bool
}

var (
	_ GraphNodeModuleInstance       = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeReferenceable        = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeReferencer           = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeConfigResource       = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeResourceInstance     = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeAttachResourceConfig = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeAttachResourceState  = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeExecutable           = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeProviderConsumer     = (*NodePlannableResourceInstanceOrphan)(nil)
)

func (n *NodePlannableResourceInstanceOrphan) Name() string {
	return n.ResourceInstanceAddr().String() + " (orphan)"
}

// GraphNodeExecutable
func (n *NodePlannableResourceInstanceOrphan) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
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

func (n *NodePlannableResourceInstanceOrphan) ProvidedBy() (addr addrs.ProviderConfig, exact bool) {
	if n.Addr.Resource.Resource.Mode == addrs.DataResourceMode {
		// indicate that this node does not require a configured provider
		return nil, true
	}
	return n.NodeAbstractResourceInstance.ProvidedBy()
}

func (n *NodePlannableResourceInstanceOrphan) dataResourceExecute(ctx EvalContext) tfdiags.Diagnostics {
	// A data source that is no longer in the config is removed from the state
	log.Printf("[TRACE] NodePlannableResourceInstanceOrphan: removing state object for %s", n.Addr)

	// we need to update both the refresh state to refresh the current data
	// source, and the working state for plan-time evaluations.
	refreshState := ctx.RefreshState()
	refreshState.SetResourceInstanceCurrent(n.Addr, nil, n.ResolvedProvider)

	workingState := ctx.State()
	workingState.SetResourceInstanceCurrent(n.Addr, nil, n.ResolvedProvider)
	return nil
}

func (n *NodePlannableResourceInstanceOrphan) managedResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	addr := n.ResourceInstanceAddr()

	oldState, readDiags := n.readResourceInstanceState(ctx, addr)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return diags
	}

	// Note any upgrades that readResourceInstanceState might've done in the
	// prevRunState, so that it'll conform to current schema.
	diags = diags.Append(n.writeResourceInstanceState(ctx, oldState, prevRunState))
	if diags.HasErrors() {
		return diags
	}
	// Also the refreshState, because that should still reflect schema upgrades
	// even if not refreshing.
	diags = diags.Append(n.writeResourceInstanceState(ctx, oldState, refreshState))
	if diags.HasErrors() {
		return diags
	}

	if !n.skipRefresh {
		// Refresh this instance even though it is going to be destroyed, in
		// order to catch missing resources. If this is a normal plan,
		// providers expect a Read request to remove missing resources from the
		// plan before apply, and may not handle a missing resource during
		// Delete correctly.  If this is a simple refresh, Terraform is
		// expected to remove the missing resource from the state entirely
		refreshedState, refreshDiags := n.refresh(ctx, states.NotDeposed, oldState)
		diags = diags.Append(refreshDiags)
		if diags.HasErrors() {
			return diags
		}

		diags = diags.Append(n.writeResourceInstanceState(ctx, refreshedState, refreshState))
		if diags.HasErrors() {
			return diags
		}

		// If we refreshed then our subsequent planning should be in terms of
		// the new object, not the original object.
		oldState = refreshedState
	}

	if !n.skipPlanChanges {
		var change *plans.ResourceInstanceChange
		change, destroyPlanDiags := n.planDestroy(ctx, oldState, "")
		diags = diags.Append(destroyPlanDiags)
		if diags.HasErrors() {
			return diags
		}

		diags = diags.Append(n.checkPreventDestroy(change))
		if diags.HasErrors() {
			return diags
		}

		diags = diags.Append(n.writeChange(ctx, change, ""))
		if diags.HasErrors() {
			return diags
		}

		diags = diags.Append(n.writeResourceInstanceState(ctx, nil, workingState))
	} else {
		// The working state should at least be updated with the result
		// of upgrading and refreshing from above.
		diags = diags.Append(n.writeResourceInstanceState(ctx, oldState, workingState))
	}

	return diags
}

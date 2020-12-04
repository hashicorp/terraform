package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// NodePlannableResourceInstanceOrphan represents a resource that is "applyable":
// it is ready to be applied and is represented by a diff.
type NodePlannableResourceInstanceOrphan struct {
	*NodeAbstractResourceInstance

	skipRefresh bool
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

func (n *NodePlannableResourceInstanceOrphan) dataResourceExecute(ctx EvalContext) tfdiags.Diagnostics {
	// A data source that is no longer in the config is removed from the state
	log.Printf("[TRACE] NodePlannableResourceInstanceOrphan: removing state object for %s", n.Addr)
	state := ctx.RefreshState()
	state.SetResourceInstanceCurrent(n.Addr, nil, n.ResolvedProvider)
	return nil
}

func (n *NodePlannableResourceInstanceOrphan) managedResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	addr := n.ResourceInstanceAddr()

	// Declare a bunch of variables that are used for state during
	// evaluation. These are written to by-address below.
	var change *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject

	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	state, err = n.ReadResourceInstanceState(ctx, addr)
	diags = diags.Append(err)
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
		refresh := &EvalRefresh{
			Addr:           addr.Resource,
			ProviderAddr:   n.ResolvedProvider,
			Provider:       &provider,
			ProviderMetas:  n.ProviderMetas,
			ProviderSchema: &providerSchema,
			State:          &state,
			Output:         &state,
		}
		diags = diags.Append(refresh.Eval(ctx))
		if diags.HasErrors() {
			return diags
		}

		diags = diags.Append(n.writeResourceInstanceState(ctx, state, n.Dependencies, refreshState))
		if diags.HasErrors() {
			return diags
		}
	}

	diffDestroy := &EvalDiffDestroy{
		Addr:         addr.Resource,
		State:        &state,
		ProviderAddr: n.ResolvedProvider,
		Output:       &change,
		OutputState:  &state, // Will point to a nil state after this complete, signalling destroyed
	}
	diags = diags.Append(diffDestroy.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(n.checkPreventDestroy(change))
	if diags.HasErrors() {
		return diags
	}

	writeDiff := &EvalWriteDiff{
		Addr:           addr.Resource,
		ProviderSchema: &providerSchema,
		Change:         &change,
	}
	diags = diags.Append(writeDiff.Eval(ctx))
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(n.writeResourceInstanceState(ctx, state, n.Dependencies, workingState))
	return diags
}

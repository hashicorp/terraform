package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
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
func (n *NodePlannableResourceInstanceOrphan) Execute(ctx EvalContext, op walkOperation) error {
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

func (n *NodePlannableResourceInstanceOrphan) dataResourceExecute(ctx EvalContext) error {
	// A data source that is no longer in the config is removed from the state
	log.Printf("[TRACE] NodePlannableResourceInstanceOrphan: removing state object for %s", n.Addr)
	state := ctx.RefreshState()
	state.SetResourceInstanceCurrent(n.Addr, nil, n.ResolvedProvider)
	return nil
}

func (n *NodePlannableResourceInstanceOrphan) managedResourceExecute(ctx EvalContext) error {
	addr := n.ResourceInstanceAddr()

	// Declare a bunch of variables that are used for state during
	// evaluation. These are written to by-address below.
	var change *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject

	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return err
	}

	state, err = n.ReadResourceInstanceState(ctx, addr)
	if err != nil {
		return err
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
		_, err = refresh.Eval(ctx)
		if err != nil {
			return err
		}

		writeRefreshState := &EvalWriteState{
			Addr:           addr.Resource,
			ProviderAddr:   n.ResolvedProvider,
			ProviderSchema: &providerSchema,
			State:          &state,
			targetState:    refreshState,
		}
		_, err = writeRefreshState.Eval(ctx)
		if err != nil {
			return err
		}
	}

	diffDestroy := &EvalDiffDestroy{
		Addr:         addr.Resource,
		State:        &state,
		ProviderAddr: n.ResolvedProvider,
		Output:       &change,
		OutputState:  &state, // Will point to a nil state after this complete, signalling destroyed
	}
	_, err = diffDestroy.Eval(ctx)
	if err != nil {
		return err
	}

	err = n.checkPreventDestroy(change)
	if err != nil {
		return err
	}

	writeDiff := &EvalWriteDiff{
		Addr:           addr.Resource,
		ProviderSchema: &providerSchema,
		Change:         &change,
	}
	_, err = writeDiff.Eval(ctx)
	if err != nil {
		return err
	}

	writeState := &EvalWriteState{
		Addr:           addr.Resource,
		ProviderAddr:   n.ResolvedProvider,
		ProviderSchema: &providerSchema,
		State:          &state,
	}
	_, err = writeState.Eval(ctx)
	if err != nil {
		return err
	}
	return nil
}

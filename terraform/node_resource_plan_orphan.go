package terraform

import (
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
)

// NodePlannableResourceInstanceOrphan represents a resource that is "applyable":
// it is ready to be applied and is represented by a diff.
type NodePlannableResourceInstanceOrphan struct {
	*NodeAbstractResourceInstance
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

	// Declare a bunch of variables that are used for state during
	// evaluation. These are written to by-address below.
	var change *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject

	_, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return err
	}

	state, err = n.ReadResourceInstanceState(ctx, addr)
	if err != nil {
		return err
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

	checkPreventDestroy := &EvalCheckPreventDestroy{
		Addr:   addr.Resource,
		Config: n.Config,
		Change: &change,
	}
	_, err = checkPreventDestroy.Eval(ctx)
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

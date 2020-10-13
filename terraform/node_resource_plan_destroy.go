package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
)

// NodePlanDestroyableResourceInstance represents a resource that is ready
// to be planned for destruction.
type NodePlanDestroyableResourceInstance struct {
	*NodeAbstractResourceInstance
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
func (n *NodePlanDestroyableResourceInstance) Execute(ctx EvalContext, op walkOperation) error {
	addr := n.ResourceInstanceAddr()

	// Declare a bunch of variables that are used for state during
	// evaluation. These are written to by address in the EvalNodes we
	// declare below.
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
		ProviderAddr: n.ResolvedProvider,
		State:        &state,
		Output:       &change,
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
	return err
}

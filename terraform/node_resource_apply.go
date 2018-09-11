package terraform

import (
	"log"
)

// NodeApplyableResource represents a resource that is "applyable":
// it may need to have its record in the state adjusted to match configuration.
//
// Unlike in the plan walk, this resource node does not DynamicExpand. Instead,
// it should be inserted into the same graph as any instances of the nodes
// with dependency edges ensuring that the resource is evaluated before any
// of its instances, which will turn ensure that the whole-resource record
// in the state is suitably prepared to receive any updates to instances.
type NodeApplyableResource struct {
	*NodeAbstractResource
}

var (
	_ GraphNodeResource             = (*NodeApplyableResource)(nil)
	_ GraphNodeEvalable             = (*NodeApplyableResource)(nil)
	_ GraphNodeProviderConsumer     = (*NodeApplyableResource)(nil)
	_ GraphNodeAttachResourceConfig = (*NodeApplyableResource)(nil)
)

func (n *NodeApplyableResource) Name() string {
	return n.NodeAbstractResource.Name() + " (prepare state)"
}

// GraphNodeEvalable
func (n *NodeApplyableResource) EvalTree() EvalNode {
	addr := n.ResourceAddr()
	config := n.Config
	providerAddr := n.ResolvedProvider

	if config == nil {
		// Nothing to do, then.
		log.Printf("[TRACE] NodeApplyableResource: no configuration present for %s", addr)
		return &EvalNoop{}
	}

	return &EvalWriteResourceState{
		Addr:         addr.Resource,
		Config:       config,
		ProviderAddr: providerAddr,
	}
}

package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
)

// NodeDestroyableDataResourceInstance represents a resource that is "destroyable":
// it is ready to be destroyed.
type NodeDestroyableDataResourceInstance struct {
	*NodeAbstractResourceInstance
}

// GraphNodeEvalable
func (n *NodeDestroyableDataResourceInstance) EvalTree() EvalNode {
	addr := n.ResourceInstanceAddr()

	var providerSchema *ProviderSchema
	// We don't need the provider, but we're calling EvalGetProvider to load the
	// schema.
	var provider providers.Interface

	// Just destroy it.
	var state *states.ResourceInstanceObject
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Addr:   n.ResolvedProvider,
				Output: &provider,
				Schema: &providerSchema,
			},
			&EvalWriteState{
				Addr:           addr.Resource,
				State:          &state,
				ProviderAddr:   n.ResolvedProvider,
				ProviderSchema: &providerSchema,
			},
		},
	}
}

func (n *NodeDestroyableDataResourceInstance) References() []*addrs.Reference {
	// We don't evaluate configuration when destroying a data resource,
	// so any references in the configuration are irrelevant. Omitting
	// these simplifies the graph slightly and can permit greater
	// concurrency of graph operations.
	return nil
}

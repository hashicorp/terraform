package terraform

import (
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
)

// nodeTransientDataResourceInstance represents a special sort of data resource
// whose result is re-read during each walk and never persisted to plan nor
// state snapshots.
//
// Transient data resources are those which have "storage = transient" inside
// the lifecycle block in their configurations.
type nodeTransientDataResourceInstance struct {
	*NodeAbstractResourceInstance
}

// GraphNodeEvalable
func (n *nodeTransientDataResourceInstance) EvalTree() EvalNode {
	addr := n.ResourceInstanceAddr()

	// These variables are the state for the eval sequence below, and are
	// updated through pointers.
	var provider providers.Interface
	var providerSchema *ProviderSchema
	var change *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalGetProvider{
				Addr:   n.ResolvedProvider,
				Output: &provider,
				Schema: &providerSchema,
			},

			&EvalReadState{
				Addr:           addr.Resource,
				Provider:       &provider,
				ProviderSchema: &providerSchema,
				Output:         &state,
			},

			// EvalReadDataTransient is a special variant of evalReadData
			// that always either produces a fully-populated result or
			// produces error diagnostics explaining why not.
			&evalReadDataTransient{
				evalReadData{
					Addr:           addr.Resource,
					Config:         n.Config,
					Provider:       &provider,
					ProviderAddr:   n.ResolvedProvider,
					ProviderMetas:  n.ProviderMetas,
					ProviderSchema: &providerSchema,
					OutputChange:   &change,
					State:          &state,
				},
			},

			&EvalWriteState{
				Addr:           addr.Resource,
				ProviderAddr:   n.ResolvedProvider,
				State:          &state,
				ProviderSchema: &providerSchema,
			},
			&EvalUpdateStateHook{},
		},
	}
}

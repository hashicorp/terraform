package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty"
)

// NodePlannableResourceInstance represents a _single_ resource
// instance that is plannable. This means this represents a single
// count index, for example.
type NodePlannableResourceInstance struct {
	*NodeAbstractResourceInstance
}

var (
	_ GraphNodeSubPath              = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeReferenceable        = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeReferencer           = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeResource             = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeResourceInstance     = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeAttachResourceConfig = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeAttachResourceState  = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeEvalable             = (*NodePlannableResourceInstance)(nil)
)

// GraphNodeEvalable
func (n *NodePlannableResourceInstance) EvalTree() EvalNode {
	addr := n.ResourceInstanceAddr()

	// State still uses legacy-style internal ids, so we need to shim to get
	// a suitable key to use.
	stateId := NewLegacyResourceInstanceAddress(addr).stateId()

	// Determine the dependencies for the state.
	stateDeps := n.StateReferences()

	// Eval info is different depending on what kind of resource this is
	switch addr.Resource.Resource.Mode {
	case addrs.ManagedResourceMode:
		return n.evalTreeManagedResource(addr, stateId, stateDeps)
	case addrs.DataResourceMode:
		return n.evalTreeDataResource(addr, stateId, stateDeps)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.Config.Mode))
	}
}

func (n *NodePlannableResourceInstance) evalTreeDataResource(addr addrs.AbsResourceInstance, stateId string, stateDeps []string) EvalNode {
	config := n.Config
	var provider ResourceProvider
	var providerSchema *ProviderSchema
	var diff *InstanceDiff
	var state *InstanceState
	var configVal cty.Value

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalReadState{
				Name:   stateId,
				Output: &state,
			},

			&EvalGetProvider{
				Addr:   n.ResolvedProvider,
				Output: &provider,
				Schema: &providerSchema,
			},

			&EvalValidateSelfRef{
				Addr:           addr.Resource,
				Config:         config.Config,
				ProviderSchema: &providerSchema,
			},

			&EvalReadDataDiff{
				Addr:           addr.Resource,
				Config:         n.Config,
				Provider:       &provider,
				ProviderSchema: &providerSchema,
				Output:         &diff,
				OutputValue:    &configVal,
				OutputState:    &state,
			},

			&EvalWriteState{
				Name:         stateId,
				ResourceType: n.Config.Type,
				Provider:     n.ResolvedProvider,
				Dependencies: stateDeps,
				State:        &state,
			},

			&EvalWriteDiff{
				Name: stateId,
				Diff: &diff,
			},
		},
	}
}

func (n *NodePlannableResourceInstance) evalTreeManagedResource(addr addrs.AbsResourceInstance, stateId string, stateDeps []string) EvalNode {
	config := n.Config
	var provider ResourceProvider
	var providerSchema *ProviderSchema
	var diff *InstanceDiff
	var state *InstanceState

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalReadState{
				Name:   stateId,
				Output: &state,
			},

			&EvalGetProvider{
				Addr:   n.ResolvedProvider,
				Output: &provider,
				Schema: &providerSchema,
			},

			&EvalValidateSelfRef{
				Addr:           addr.Resource,
				Config:         config.Config,
				ProviderSchema: &providerSchema,
			},

			&EvalDiff{
				Addr:           addr.Resource,
				Config:         n.Config,
				Provider:       &provider,
				ProviderSchema: &providerSchema,
				State:          &state,
				OutputDiff:     &diff,
				OutputState:    &state,
			},
			&EvalCheckPreventDestroy{
				Addr:   addr.Resource,
				Config: n.Config,
				Diff:   &diff,
			},
			&EvalWriteState{
				Name:         stateId,
				ResourceType: n.Config.Type,
				Provider:     n.ResolvedProvider,
				Dependencies: stateDeps,
				State:        &state,
			},
			&EvalWriteDiff{
				Name: stateId,
				Diff: &diff,
			},
		},
	}
}

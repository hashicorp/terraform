package terraform

import (
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
)

// NodePlannableResourceInstanceOrphan represents a resource that is "applyable":
// it is ready to be applied and is represented by a diff.
type NodePlannableResourceInstanceOrphan struct {
	*NodeAbstractResourceInstance
}

var (
	_ GraphNodeSubPath              = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeReferenceable        = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeReferencer           = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeResource             = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeResourceInstance     = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeAttachResourceConfig = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeAttachResourceState  = (*NodePlannableResourceInstanceOrphan)(nil)
	_ GraphNodeEvalable             = (*NodePlannableResourceInstanceOrphan)(nil)
)

var (
	_ GraphNodeEvalable = (*NodePlannableResourceInstanceOrphan)(nil)
)

func (n *NodePlannableResourceInstanceOrphan) Name() string {
	return n.ResourceInstanceAddr().String() + " (orphan)"
}

// GraphNodeEvalable
func (n *NodePlannableResourceInstanceOrphan) EvalTree() EvalNode {
	addr := n.ResourceInstanceAddr()

	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var change *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject
	var provider providers.Interface
	var providerSchema *ProviderSchema

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

				Output: &state,
			},
			&EvalDiffDestroy{
				Addr:        addr.Resource,
				State:       &state,
				Output:      &change,
				OutputState: &state, // Will point to a nil state after this complete, signalling destroyed
			},
			&EvalCheckPreventDestroy{
				Addr:   addr.Resource,
				Config: n.Config,
				Change: &change,
			},
			&EvalWriteDiff{
				Addr:           addr.Resource,
				ProviderSchema: &providerSchema,
				Change:         &change,
			},
			&EvalWriteState{
				Addr:           addr.Resource,
				ProviderAddr:   n.ResolvedProvider,
				ProviderSchema: &providerSchema,
				State:          &state,
			},
		},
	}
}

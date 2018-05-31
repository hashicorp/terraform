package terraform

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

	// State still uses legacy-style internal ids, so we need to shim to get
	// a suitable key to use.
	stateId := NewLegacyResourceInstanceAddress(addr).stateId()

	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var diff *InstanceDiff
	var state *InstanceState

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalReadState{
				Name:   stateId,
				Output: &state,
			},
			&EvalDiffDestroy{
				Addr:   addr.Resource,
				State:  &state, // Will point to a nil state after this complete, signalling destroyed
				Output: &diff,
			},
			&EvalCheckPreventDestroy{
				Addr:   addr.Resource,
				Config: n.Config,
				Diff:   &diff,
			},
			&EvalWriteDiff{
				Name: stateId,
				Diff: &diff,
			},
			&EvalWriteState{
				Name:         stateId,
				ResourceType: addr.Resource.Resource.Type,
				Provider:     n.ResolvedProvider,
				State:        &state,
			},
		},
	}
}

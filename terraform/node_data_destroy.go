package terraform

// NodeDestroyableDataResource represents a resource that is "plannable":
// it is ready to be planned in order to create a diff.
type NodeDestroyableDataResource struct {
	*NodeAbstractResource
}

// GraphNodeEvalable
func (n *NodeDestroyableDataResource) EvalTree() EvalNode {
	addr := n.NodeAbstractResource.Addr

	// stateId is the ID to put into the state
	stateId := addr.stateId()

	// Just destroy it.
	var state *InstanceState
	return &EvalWriteState{
		Name:  stateId,
		State: &state, // state is nil here
	}
}

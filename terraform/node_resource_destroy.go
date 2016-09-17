package terraform

import (
	"fmt"
)

// NodeDestroyResource represents a resource that is to be destroyed.
type NodeDestroyResource struct {
	Addr          *ResourceAddress // Addr is the address for this resource
	ResourceState *ResourceState   // State is the resource state for this resource
}

func (n *NodeDestroyResource) Name() string {
	return n.Addr.String()
}

// GraphNodeSubPath
func (n *NodeDestroyResource) Path() []string {
	return n.Addr.Path
}

// GraphNodeProviderConsumer
func (n *NodeDestroyResource) ProvidedBy() []string {
	// If we have state, then we will use the provider from there
	if n.ResourceState != nil && n.ResourceState.Provider != "" {
		return []string{n.ResourceState.Provider}
	}

	// Use our type
	return []string{resourceProvider(n.Addr.Type, "")}
}

// GraphNodeAttachResourceState
func (n *NodeDestroyResource) ResourceAddr() *ResourceAddress {
	return n.Addr
}

// GraphNodeAttachResourceState
func (n *NodeDestroyResource) AttachResourceState(s *ResourceState) {
	n.ResourceState = s
}

// GraphNodeEvalable
func (n *NodeDestroyResource) EvalTree() EvalNode {
	// stateId is the ID to put into the state
	stateId := n.Addr.stateId()
	if n.Addr.Index > -1 {
		stateId = fmt.Sprintf("%s.%d", stateId, n.Addr.Index)
	}

	// Build the instance info. More of this will be populated during eval
	info := &InstanceInfo{
		Id:   stateId,
		Type: n.Addr.Type,
	}

	// Get our state
	rs := n.ResourceState
	if rs == nil {
		rs = &ResourceState{}
	}
	rs.Provider = n.ProvidedBy()[0]

	var diffApply *InstanceDiff
	var provider ResourceProvider
	var state *InstanceState
	var err error
	return &EvalOpFilter{
		Ops: []walkOperation{walkApply, walkDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				// Get the saved diff for apply
				&EvalReadDiff{
					Name: stateId,
					Diff: &diffApply,
				},

				// Filter the diff so we only get the destroy
				&EvalFilterDiff{
					Diff:    &diffApply,
					Output:  &diffApply,
					Destroy: true,
				},

				// If we're not destroying, then compare diffs
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						if diffApply != nil && diffApply.GetDestroy() {
							return true, nil
						}

						return true, EvalEarlyExitError{}
					},
					Then: EvalNoop{},
				},

				// Load the instance info so we have the module path set
				&EvalInstanceInfo{Info: info},

				&EvalGetProvider{
					Name:   n.ProvidedBy()[0],
					Output: &provider,
				},
				&EvalReadState{
					Name:   stateId,
					Output: &state,
				},
				&EvalRequireState{
					State: &state,
				},
				// Make sure we handle data sources properly.
				&EvalIf{
					If: func(ctx EvalContext) (bool, error) {
						/* TODO: data source
						if n.Resource.Mode == config.DataResourceMode {
							return true, nil
						}
						*/

						return false, nil
					},

					Then: &EvalReadDataApply{
						Info:     info,
						Diff:     &diffApply,
						Provider: &provider,
						Output:   &state,
					},
					Else: &EvalApply{
						Info:     info,
						State:    &state,
						Diff:     &diffApply,
						Provider: &provider,
						Output:   &state,
						Error:    &err,
					},
				},
				&EvalWriteState{
					Name:         stateId,
					ResourceType: n.Addr.Type,
					Provider:     rs.Provider,
					Dependencies: rs.Dependencies,
					State:        &state,
				},
				&EvalApplyPost{
					Info:  info,
					State: &state,
					Error: &err,
				},
			},
		},
	}
}

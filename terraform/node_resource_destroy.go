package terraform

import (
	"fmt"
)

// NodeDestroyResource represents a resource that is to be destroyed.
type NodeDestroyResource struct {
	*NodeAbstractResource
}

func (n *NodeDestroyResource) Name() string {
	return n.NodeAbstractResource.Name() + " (destroy)"
}

// GraphNodeDestroyer
func (n *NodeDestroyResource) DestroyAddr() *ResourceAddress {
	return n.Addr
}

// GraphNodeDestroyerCBD
func (n *NodeDestroyResource) CreateBeforeDestroy() bool {
	// If we have no config, we just assume no
	if n.Config == nil {
		return false
	}

	return n.Config.Lifecycle.CreateBeforeDestroy
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
				&EvalUpdateStateHook{},
			},
		},
	}
}

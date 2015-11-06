package terraform

import "fmt"

// DeposedTransformer is a GraphTransformer that adds deposed resources
// to the graph.
type DeposedTransformer struct {
	// State is the global state. We'll automatically find the correct
	// ModuleState based on the Graph.Path that is being transformed.
	State *State

	// View, if non-empty, is the ModuleState.View used around the state
	// to find deposed resources.
	View string
}

func (t *DeposedTransformer) Transform(g *Graph) error {
	state := t.State.ModuleByPath(g.Path)
	if state == nil {
		// If there is no state for our module there can't be any deposed
		// resources, since they live in the state.
		return nil
	}

	// If we have a view, apply it now
	if t.View != "" {
		state = state.View(t.View)
	}

	// Go through all the resources in our state to look for deposed resources
	for k, rs := range state.Resources {
		// If we have no deposed resources, then move on
		if len(rs.Deposed) == 0 {
			continue
		}
		deposed := rs.Deposed

		for i, _ := range deposed {
			g.Add(&graphNodeDeposedResource{
				Index:        i,
				ResourceName: k,
				ResourceType: rs.Type,
				Provider:     rs.Provider,
			})
		}
	}

	return nil
}

// graphNodeDeposedResource is the graph vertex representing a deposed resource.
type graphNodeDeposedResource struct {
	Index        int
	ResourceName string
	ResourceType string
	Provider     string
}

func (n *graphNodeDeposedResource) Name() string {
	return fmt.Sprintf("%s (deposed #%d)", n.ResourceName, n.Index)
}

func (n *graphNodeDeposedResource) ProvidedBy() []string {
	return []string{resourceProvider(n.ResourceName, n.Provider)}
}

// GraphNodeEvalable impl.
func (n *graphNodeDeposedResource) EvalTree() EvalNode {
	var provider ResourceProvider
	var state *InstanceState

	seq := &EvalSequence{Nodes: make([]EvalNode, 0, 5)}

	// Build instance info
	info := &InstanceInfo{Id: n.ResourceName, Type: n.ResourceType}
	seq.Nodes = append(seq.Nodes, &EvalInstanceInfo{Info: info})

	// Refresh the resource
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Name:   n.ProvidedBy()[0],
					Output: &provider,
				},
				&EvalReadStateDeposed{
					Name:   n.ResourceName,
					Output: &state,
					Index:  n.Index,
				},
				&EvalRefresh{
					Info:     info,
					Provider: &provider,
					State:    &state,
					Output:   &state,
				},
				&EvalWriteStateDeposed{
					Name:         n.ResourceName,
					ResourceType: n.ResourceType,
					Provider:     n.Provider,
					State:        &state,
					Index:        n.Index,
				},
			},
		},
	})

	// Apply
	var diff *InstanceDiff
	var err error
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkApply, walkDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Name:   n.ProvidedBy()[0],
					Output: &provider,
				},
				&EvalReadStateDeposed{
					Name:   n.ResourceName,
					Output: &state,
					Index:  n.Index,
				},
				&EvalDiffDestroy{
					Info:   info,
					State:  &state,
					Output: &diff,
				},
				&EvalApply{
					Info:     info,
					State:    &state,
					Diff:     &diff,
					Provider: &provider,
					Output:   &state,
					Error:    &err,
				},
				// Always write the resource back to the state deposed... if it
				// was successfully destroyed it will be pruned. If it was not, it will
				// be caught on the next run.
				&EvalWriteStateDeposed{
					Name:         n.ResourceName,
					ResourceType: n.ResourceType,
					Provider:     n.Provider,
					State:        &state,
					Index:        n.Index,
				},
				&EvalReturnError{
					Error: &err,
				},
				&EvalUpdateStateHook{},
			},
		},
	})

	return seq
}

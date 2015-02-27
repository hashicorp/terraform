package terraform

import "fmt"

// DeposedTransformer is a GraphTransformer that adds tainted resources
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
		// If there is no state for our module there can't be any tainted
		// resources, since they live in the state.
		return nil
	}

	// If we have a view, apply it now
	if t.View != "" {
		state = state.View(t.View)
	}

	// Go through all the resources in our state to look for tainted resources
	for k, rs := range state.Resources {
		if rs.Deposed == nil {
			continue
		}

		g.Add(&graphNodeDeposedResource{
			ResourceName: k,
			ResourceType: rs.Type,
		})
	}

	return nil
}

// graphNodeDeposedResource is the graph vertex representing a deposed resource.
type graphNodeDeposedResource struct {
	ResourceName string
	ResourceType string
}

func (n *graphNodeDeposedResource) Name() string {
	return fmt.Sprintf("%s (deposed)", n.ResourceName)
}

func (n *graphNodeDeposedResource) ProvidedBy() []string {
	return []string{resourceProvider(n.ResourceName)}
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
				},
				&EvalRefresh{
					Info:     info,
					Provider: &provider,
					State:    &state,
					Output:   &state,
				},
				&EvalWriteState{
					Name:         n.ResourceName,
					ResourceType: n.ResourceType,
					State:        &state,
					Deposed:      true,
				},
			},
		},
	})

	// Apply
	var diff *InstanceDiff
	var err error
	var emptyState *InstanceState
	tainted := true
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkApply},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Name:   n.ProvidedBy()[0],
					Output: &provider,
				},
				&EvalReadStateDeposed{
					Name:   n.ResourceName,
					Output: &state,
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
				// Always write the resource back to the state tainted... if it
				// successfully destroyed it will be pruned. If it did not, it will
				// remain tainted.
				&EvalWriteState{
					Name:         n.ResourceName,
					ResourceType: n.ResourceType,
					State:        &state,
					Tainted:      &tainted,
					TaintedIndex: -1,
				},
				// Then clear the deposed state.
				&EvalWriteState{
					Name:         n.ResourceName,
					ResourceType: n.ResourceType,
					State:        &emptyState,
					Deposed:      true,
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

package terraform

import (
	"fmt"
)

// TraintedTransformer is a GraphTransformer that adds tainted resources
// to the graph.
type TaintedTransformer struct {
	// State is the global state. We'll automatically find the correct
	// ModuleState based on the Graph.Path that is being transformed.
	State *State

	// View, if non-empty, is the ModuleState.View used around the state
	// to find tainted resources.
	View string
}

func (t *TaintedTransformer) Transform(g *Graph) error {
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
		// If we have no tainted resources, then move on
		if len(rs.Tainted) == 0 {
			continue
		}

		for i, _ := range rs.Tainted {
			// Add the graph node and make the connection from any untainted
			// resources with this name to the tainted resource, so that
			// the tainted resource gets destroyed first.
			g.Add(&graphNodeTaintedResource{
				Index:        i,
				ResourceName: k,
				ResourceType: rs.Type,
			})
		}
	}

	return nil
}

// graphNodeTaintedResource is the graph vertex representing a tainted resource.
type graphNodeTaintedResource struct {
	Index        int
	ResourceName string
	ResourceType string
}

func (n *graphNodeTaintedResource) Name() string {
	return fmt.Sprintf("%s (tainted #%d)", n.ResourceName, n.Index+1)
}

func (n *graphNodeTaintedResource) ProvidedBy() []string {
	return []string{resourceProvider(n.ResourceName)}
}

// GraphNodeEvalable impl.
func (n *graphNodeTaintedResource) EvalTree() EvalNode {
	var state *InstanceState
	tainted := true

	seq := &EvalSequence{Nodes: make([]EvalNode, 0, 5)}

	// Build instance info
	info := &InstanceInfo{Id: n.ResourceName, Type: n.ResourceType}
	seq.Nodes = append(seq.Nodes, &EvalInstanceInfo{Info: info})

	// Refresh the resource
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalReadState{
					Name:         n.ResourceName,
					Tainted:      true,
					TaintedIndex: n.Index,
					Output:       &state,
				},
				&EvalRefresh{
					Info:     info,
					Provider: &EvalGetProvider{Name: n.ProvidedBy()[0]},
					State:    &state,
					Output:   &state,
				},
				&EvalWriteState{
					Name:         n.ResourceName,
					ResourceType: n.ResourceType,
					State:        &state,
					Tainted:      &tainted,
					TaintedIndex: n.Index,
				},
			},
		},
	})

	// Apply
	var provider ResourceProvider
	var diff *InstanceDiff
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkApply},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Name:   n.ProvidedBy()[0],
					Output: &provider,
				},
				&EvalReadState{
					Name:         n.ResourceName,
					Tainted:      true,
					TaintedIndex: n.Index,
					Output:       &state,
				},
				&EvalDiffDestroy{
					Info: info,
					State: &EvalReadState{
						Name:         n.ResourceName,
						Tainted:      true,
						TaintedIndex: n.Index,
					},
					Output: &diff,
				},
				&EvalApply{
					Info:     info,
					State:    &state,
					Diff:     &diff,
					Provider: &provider,
					Output:   &state,
				},
				&EvalWriteState{
					Name:         n.ResourceName,
					ResourceType: n.ResourceType,
					State:        &state,
					Tainted:      &tainted,
					TaintedIndex: n.Index,
				},
			},
		},
	})

	return seq
}

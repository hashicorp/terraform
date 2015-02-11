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
}

func (t *TaintedTransformer) Transform(g *Graph) error {
	state := t.State.ModuleByPath(g.Path)
	if state == nil {
		// If there is no state for our module there can't be any tainted
		// resources, since they live in the state.
		return nil
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
			g.ConnectFrom(k, g.Add(&graphNodeTaintedResource{
				Index:        i,
				ResourceName: k,
				ResourceType: rs.Type,
			}))
		}
	}

	// TODO: Any other dependencies?

	return nil
}

// graphNodeTaintedResource is the graph vertex representing a tainted resource.
type graphNodeTaintedResource struct {
	Index        int
	ResourceName string
	ResourceType string
}

func (n *graphNodeTaintedResource) DependentOn() []string {
	return []string{n.ResourceName}
}

func (n *graphNodeTaintedResource) Name() string {
	return fmt.Sprintf("%s (tainted #%d)", n.ResourceName, n.Index+1)
}

func (n *graphNodeTaintedResource) ProvidedBy() []string {
	return []string{resourceProvider(n.ResourceName)}
}

// GraphNodeEvalable impl.
func (n *graphNodeTaintedResource) EvalTree() EvalNode {
	seq := &EvalSequence{Nodes: make([]EvalNode, 0, 5)}

	// Build instance info
	info := &InstanceInfo{Id: n.ResourceName, Type: n.ResourceType}
	seq.Nodes = append(seq.Nodes, &EvalInstanceInfo{Info: info})

	// Refresh the resource
	seq.Nodes = append(seq.Nodes, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh},
		Node: &EvalWriteState{
			Name:         n.ResourceName,
			ResourceType: n.ResourceType,
			Dependencies: n.DependentOn(),
			Tainted:      true,
			TaintedIndex: n.Index,
			State: &EvalRefresh{
				Info:     info,
				Provider: &EvalGetProvider{Name: n.ProvidedBy()[0]},
				State: &EvalReadState{
					Name:         n.ResourceName,
					Tainted:      true,
					TaintedIndex: n.Index,
				},
			},
		},
	})

	return seq
}

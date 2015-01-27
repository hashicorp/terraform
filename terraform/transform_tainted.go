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
			g.Add(&graphNodeTaintedResource{
				Index:        i,
				ResourceName: k,
			})
		}
	}

	// TODO: Dependencies?

	return nil
}

// graphNodeTaintedResource is the graph vertex representing a tainted resource.
type graphNodeTaintedResource struct {
	Index        int
	ResourceName string
}

func (n *graphNodeTaintedResource) DependableName() []string {
	return []string{n.dependableName()}
}

func (n *graphNodeTaintedResource) Name() string {
	return fmt.Sprintf("%s (tainted #%d)", n.ResourceName, n.Index+1)
}

func (n *graphNodeTaintedResource) dependableName() string {
	return n.ResourceName
}

package terraform

import (
	"fmt"
)

// GraphNodeOutput is an interface that nodes that are outputs must
// implement. The OutputName returned is the name of the output key
// that they manage.
type GraphNodeOutput interface {
	OutputName() string
}

// AddOutputOrphanTransformer is a transformer that adds output orphans
// to the graph. Output orphans are outputs that are no longer in the
// configuration and therefore need to be removed from the state.
type AddOutputOrphanTransformer struct {
	State *State
}

func (t *AddOutputOrphanTransformer) Transform(g *Graph) error {
	// Get the state for this module. If we have no state, we have no orphans
	state := t.State.ModuleByPath(g.Path)
	if state == nil {
		return nil
	}

	// Create the set of outputs we do have in the graph
	found := make(map[string]struct{})
	for _, v := range g.Vertices() {
		on, ok := v.(GraphNodeOutput)
		if !ok {
			continue
		}

		found[on.OutputName()] = struct{}{}
	}

	// Go over all the outputs. If we don't have a graph node for it,
	// create it. It doesn't need to depend on anything, since its just
	// setting it empty.
	for k, _ := range state.Outputs {
		if _, ok := found[k]; ok {
			continue
		}

		g.Add(&graphNodeOrphanOutput{OutputName: k})
	}

	return nil
}

type graphNodeOrphanOutput struct {
	OutputName string
}

func (n *graphNodeOrphanOutput) Name() string {
	return fmt.Sprintf("output.%s (orphan)", n.OutputName)
}

package terraform

// ModuleDestroyTransformer is a GraphTransformer that adds a node
// to the graph that will add a module destroy node for all modules in
// the state.
//
// NOTE: This is _completely unnecessary_ in the new graph worlds. This is
// only done to make old tests pass. However, this node does nothing in
// the new apply graph.
type ModuleDestroyTransformer struct {
	State *State
}

func (t *ModuleDestroyTransformer) Transform(g *Graph) error {
	// If empty do nothing
	if t.State.Empty() {
		return nil
	}

	for _, ms := range t.State.Modules {
		// Just a silly edge case that is required to get old tests to pass.
		// It is probably a bug with the old graph but we mimic it here
		// so that old tests pass.
		if len(ms.Path) <= 1 {
			continue
		}

		// Create the node
		n := &NodeDestroyableModuleVariable{PathValue: ms.Path}

		// Add it to the graph. We don't need any edges because
		// it can happen whenever.
		g.Add(n)
	}

	return nil
}

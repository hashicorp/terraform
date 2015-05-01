package terraform

// GraphNodeFlattenable must be implemented by nodes that can be flattened
// into the graph.
type GraphNodeFlattenable interface {
	GraphNodeSubgraph

	// Flatten should return true if this should be flattened.
	Flatten() bool
}

// FlattenTransform is a transformer that goes through the graph, finds
// subgraphs that can be flattened, and flattens them into this graph,
// removing the prior subgraph node.
type FlattenTransform struct{}

func (t *FlattenTransform) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		fn, ok := v.(GraphNodeFlattenable)
		if !ok {
			continue
		}

		// If we don't want to be flattened, don't do it
		if !fn.Flatten() {
			continue
		}

		// Get the subgraph and flatten it into this one
		subgraph := fn.Subgraph()
		for _, sv := range subgraph.Vertices() {
			g.Add(sv)
		}
		for _, se := range subgraph.Edges() {
			g.Connect(se)
		}

		// Remove the old node
		g.Remove(v)
	}

	return nil
}

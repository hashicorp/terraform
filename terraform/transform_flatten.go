package terraform

// GraphNodeFlattenable must be implemented by nodes that can be flattened
// into the graph.
type GraphNodeFlattenable interface {
	FlattenGraph() *Graph
}

// FlattenTransformer is a transformer that goes through the graph, finds
// subgraphs that can be flattened, and flattens them into this graph,
// removing the prior subgraph node.
type FlattenTransformer struct{}

func (t *FlattenTransformer) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		fn, ok := v.(GraphNodeFlattenable)
		if !ok {
			continue
		}

		// If we don't want to be flattened, don't do it
		subgraph := fn.FlattenGraph()
		if subgraph == nil {
			continue
		}

		// Flatten the subgraph into this one. Keep any existing
		// connections that existed.
		for _, sv := range subgraph.Vertices() {
			g.Add(sv)
		}
		for _, se := range subgraph.Edges() {
			g.Connect(se)
		}

		// Remove the old node
		g.Remove(v)

		// Connect the dependencies for all the new nodes that we added.
		// This will properly connect variables to their sources, for example.
		for _, sv := range subgraph.Vertices() {
			g.ConnectDependent(sv)
		}
	}

	return nil
}

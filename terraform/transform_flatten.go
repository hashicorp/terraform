package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

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

		// Get all the things that depend on this node. We'll re-connect
		// dependents later. We have to copy these here since the UpEdges
		// value will be deleted after the Remove below.
		dependents := make([]dag.Vertex, 0, 5)
		for _, v := range g.UpEdges(v).List() {
			dependents = append(dependents, v)
		}

		// Remove the old node
		g.Remove(v)

		// Flatten the subgraph into this one. Keep any existing
		// connections that existed.
		for _, sv := range subgraph.Vertices() {
			g.Add(sv)
		}
		for _, se := range subgraph.Edges() {
			g.Connect(se)
		}

		// Connect the dependencies for all the new nodes that we added.
		// This will properly connect variables to their sources, for example.
		for _, sv := range subgraph.Vertices() {
			g.ConnectDependent(sv)
		}

		// Re-connect all the things that dependend on the graph
		// we just flattened. This should connect them back into the
		// correct nodes if their DependentOn() is setup correctly.
		for _, v := range dependents {
			g.ConnectDependent(v)
		}
	}

	return nil
}

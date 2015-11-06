package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/dag"
)

// GraphNodeFlatGraph must be implemented by nodes that have subgraphs
// that they want flattened into the graph.
type GraphNodeFlatGraph interface {
	FlattenGraph() *Graph
}

// GraphNodeFlattenable must be implemented by all nodes that can be
// flattened. If a FlattenGraph returns any nodes that can't be flattened,
// it will be an error.
//
// If Flatten returns nil for the Vertex along with a nil error, it will
// removed from the graph.
type GraphNodeFlattenable interface {
	Flatten(path []string) (dag.Vertex, error)
}

// FlattenTransformer is a transformer that goes through the graph, finds
// subgraphs that can be flattened, and flattens them into this graph,
// removing the prior subgraph node.
type FlattenTransformer struct{}

func (t *FlattenTransformer) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		fn, ok := v.(GraphNodeFlatGraph)
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

		// Go through the subgraph and flatten all the nodes
		for _, sv := range subgraph.Vertices() {
			// If the vertex already has a subpath then we assume it has
			// already been flattened. Ignore it.
			if _, ok := sv.(GraphNodeSubPath); ok {
				continue
			}

			fn, ok := sv.(GraphNodeFlattenable)
			if !ok {
				return fmt.Errorf(
					"unflattenable node: %s %T",
					dag.VertexName(sv), sv)
			}

			v, err := fn.Flatten(subgraph.Path)
			if err != nil {
				return fmt.Errorf(
					"error flattening %s (%T): %s",
					dag.VertexName(sv), sv, err)
			}

			if v == nil {
				subgraph.Remove(v)
			} else {
				subgraph.Replace(sv, v)
			}
		}

		// Now that we've handled any changes to the graph that are
		// needed, we can add them all to our graph along with their edges.
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

		// Re-connect all the things that dependent on the graph
		// we just flattened. This should connect them back into the
		// correct nodes if their DependentOn() is setup correctly.
		for _, v := range dependents {
			g.ConnectDependent(v)
		}
	}

	return nil
}

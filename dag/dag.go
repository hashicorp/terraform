package dag

import (
	"fmt"
)

// AcyclicGraph is a specialization of Graph that cannot have cycles. With
// this property, we get the property of sane graph traversal.
type AcyclicGraph struct {
	Graph
}

// WalkFunc is the callback used for walking the graph.
type WalkFunc func(Vertex)

// Root returns the root of the DAG, or an error.
//
// Complexity: O(V)
func (g *AcyclicGraph) Root() (Vertex, error) {
	roots := make([]Vertex, 0, 1)
	for _, v := range g.Vertices() {
		if g.UpEdges(v).Len() == 0 {
			roots = append(roots, v)
		}
	}

	if len(roots) > 1 {
		// TODO(mitchellh): make this error message a lot better
		return nil, fmt.Errorf("multiple roots: %#v", roots)
	}

	if len(roots) == 0 {
		return nil, fmt.Errorf("no roots found")
	}

	return roots[0], nil
}

// Walk walks the graph, calling your callback as each node is visited.
func (g *AcyclicGraph) Walk(cb WalkFunc) {
}

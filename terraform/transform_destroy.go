package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeDestroyable is the interface that nodes that can be destroyed
// must implement. This is used to automatically handle the creation of
// destroy nodes in the graph and the dependency ordering of those destroys.
type GraphNodeDestroyable interface {
	// DestroyNode returns the node used for the destroy. This vertex
	// should not be in the graph yet.
	DestroyNode() dag.Vertex
}

// DestroyTransformer is a GraphTransformer that creates the destruction
// nodes for things that _might_ be destroyed.
type DestroyTransformer struct{}

func (t *DestroyTransformer) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		// If it is not a destroyable, we don't care
		dn, ok := v.(GraphNodeDestroyable)
		if !ok {
			continue
		}

		// Grab the destroy side of the node and connect it through
		n := dn.DestroyNode()
		if n == nil {
			continue
		}

		// Add it to the graph
		g.Add(n)

		// Inherit all the edges from the old node
		downEdges := g.DownEdges(v).List()
		for _, edgeRaw := range downEdges {
			g.Connect(dag.BasicEdge(n, edgeRaw.(dag.Vertex)))
		}

		// Remove all the edges from the old now
		for _, edgeRaw := range downEdges {
			g.RemoveEdge(dag.BasicEdge(v, edgeRaw.(dag.Vertex)))
		}

		// Add a new edge to connect the node to be created to
		// the destroy node.
		g.Connect(dag.BasicEdge(v, n))
	}

	return nil
}

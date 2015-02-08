package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/dag"
)

// GraphNodeExapndable is an interface that nodes can implement to
// signal that they can be expanded. Expanded nodes turn into
// GraphNodeSubgraph nodes within the graph.
type GraphNodeExpandable interface {
	Expand(GraphBuilder) (*Graph, error)
}

// GraphNodeSubgraph is an interface a node can implement if it has
// a larger subgraph that should be walked.
type GraphNodeSubgraph interface {
	Subgraph() *Graph
}

// ExpandTransform is a transformer that does a subgraph expansion
// at graph transform time (vs. at eval time). The benefit of earlier
// subgraph expansion is that errors with the graph build can be detected
// at an earlier stage.
type ExpandTransform struct {
	Builder GraphBuilder
}

func (t *ExpandTransform) Transform(v dag.Vertex) (dag.Vertex, error) {
	ev, ok := v.(GraphNodeExpandable)
	if !ok {
		// This isn't an expandable vertex, so just ignore it.
		return v, nil
	}

	// Expand the subgraph!
	g, err := ev.Expand(t.Builder)
	if err != nil {
		return nil, err
	}

	// Replace with our special node
	return &graphNodeExpanded{
		Graph:        g,
		OriginalName: dag.VertexName(v),
	}, nil
}

type graphNodeExpanded struct {
	Graph        *Graph
	OriginalName string
}

func (n *graphNodeExpanded) Name() string {
	return fmt.Sprintf("%s (expanded)", n.OriginalName)
}

func (n *graphNodeExpanded) Subgraph() *Graph {
	return n.Graph
}

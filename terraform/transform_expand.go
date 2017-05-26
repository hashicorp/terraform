package terraform

import (
	"log"

	"github.com/hashicorp/terraform/dag"
)

// GraphNodeExapndable is an interface that nodes can implement to
// signal that they can be expanded. Expanded nodes turn into
// GraphNodeSubgraph nodes within the graph.
type GraphNodeExpandable interface {
	Expand(GraphBuilder) (GraphNodeSubgraph, error)
}

// GraphNodeDynamicExpandable is an interface that nodes can implement
// to signal that they can be expanded at eval-time (hence dynamic).
// These nodes are given the eval context and are expected to return
// a new subgraph.
type GraphNodeDynamicExpandable interface {
	DynamicExpand(EvalContext) (*Graph, error)
}

// GraphNodeSubgraph is an interface a node can implement if it has
// a larger subgraph that should be walked.
type GraphNodeSubgraph interface {
	Subgraph() dag.Grapher
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
	log.Printf("[DEBUG] vertex %q: static expanding", dag.VertexName(ev))
	return ev.Expand(t.Builder)
}

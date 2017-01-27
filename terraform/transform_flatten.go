package terraform

import (
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

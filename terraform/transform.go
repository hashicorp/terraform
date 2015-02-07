package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

// GraphTransformer is the interface that transformers implement. This
// interface is only for transforms that need entire graph visibility.
type GraphTransformer interface {
	Transform(*Graph) error
}

// GraphVertexTransformer is an interface that transforms a single
// Vertex within with graph. This is a specialization of GraphTransformer
// that makes it easy to do vertex replacement.
//
// The GraphTransformer that runs through the GraphVertexTransformers is
// VertexTransformer.
type GraphVertexTransformer interface {
	Transform(dag.Vertex) (dag.Vertex, error)
}

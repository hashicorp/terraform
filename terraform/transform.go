package terraform

import (
	"log"

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

// GraphTransformIf is a helper function that conditionally returns a
// GraphTransformer given. This is useful for calling inline a sequence
// of transforms without having to split it up into multiple append() calls.
func GraphTransformIf(f func() bool, then GraphTransformer) GraphTransformer {
	if f() {
		return then
	}

	return nil
}

type graphTransformerMulti struct {
	Transforms []GraphTransformer
}

func (t *graphTransformerMulti) Transform(g *Graph) error {
	for _, t := range t.Transforms {
		if err := t.Transform(g); err != nil {
			return err
		}
		log.Printf(
			"[TRACE] Graph after step %T:\n\n%s",
			t, g.StringWithNodeTypes())
	}

	return nil
}

// GraphTransformMulti combines multiple graph transformers into a single
// GraphTransformer that runs all the individual graph transformers.
func GraphTransformMulti(ts ...GraphTransformer) GraphTransformer {
	return &graphTransformerMulti{Transforms: ts}
}

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
	var lastStepStr string
	for _, t := range t.Transforms {
		log.Printf("[TRACE] (graphTransformerMulti) Executing graph transform %T", t)
		if err := t.Transform(g); err != nil {
			return err
		}
		if thisStepStr := g.StringWithNodeTypes(); thisStepStr != lastStepStr {
			log.Printf("[TRACE] (graphTransformerMulti) Completed graph transform %T with new graph:\n%s------", t, thisStepStr)
			lastStepStr = thisStepStr
		} else {
			log.Printf("[TRACE] (graphTransformerMulti) Completed graph transform %T (no changes)", t)
		}
	}

	return nil
}

// GraphTransformMulti combines multiple graph transformers into a single
// GraphTransformer that runs all the individual graph transformers.
func GraphTransformMulti(ts ...GraphTransformer) GraphTransformer {
	return &graphTransformerMulti{Transforms: ts}
}

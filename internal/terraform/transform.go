// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/logging"
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
			log.Printf("[TRACE] (graphTransformerMulti) Completed graph transform %T with new graph:\n%s  ------", t, logging.Indent(thisStepStr))
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

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/dag"
)

// TransformFilter is a GraphTransformer that filters out nodes from the graph based on a provided function. The Keep function should return true for nodes that should be kept in the graph, and false for nodes that should be removed. The transformer will mark all nodes that the node to keep depends on as well, ensuring that the resulting graph is still valid.
type TransformFilter struct {
	Keep func(node dag.Vertex) bool
}

var _ GraphTransformer = (*TransformFilter)(nil)

func (t *TransformFilter) Transform(g *Graph) error {
	// Partition vertices into kept and candidates for removal.
	var kept []dag.Vertex
	var removalCandidates []dag.Vertex
	for _, v := range g.Vertices() {
		if t.Keep(v) {
			kept = append(kept, v)
		} else {
			removalCandidates = append(removalCandidates, v)
		}
	}

	// Also keep all ancestors (transitive dependencies) of the kept
	// nodes so the resulting graph stays valid.
	ancestors := g.Ancestors(kept...)

	// Remove every vertex that isn't explicitly kept and isn't an
	// ancestor of a kept node.
	for _, v := range removalCandidates {
		if !ancestors.Include(v) {
			g.Remove(v)
		}
	}

	return nil
}

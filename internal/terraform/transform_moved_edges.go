// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/refactoring"
)

// MovedBlockEdgeTransformer adds ordering edges between moved statements.
type MovedBlockEdgeTransformer struct {
	Policy refactoring.MoveOrderingPolicy
}

type moveOrderableGraphNode interface {
	dag.Vertex
	MoveOrderingStatement() *refactoring.MoveStatement
}

func (t *MovedBlockEdgeTransformer) Transform(g *Graph) error {
	policy := t.Policy
	if policy == nil {
		policy = refactoring.DefaultMoveOrderingPolicy{}
	}

	var movedNodes []moveOrderableGraphNode
	for _, v := range g.Vertices() {
		if node, ok := v.(moveOrderableGraphNode); ok {
			movedNodes = append(movedNodes, node)
		}
	}

	for _, depender := range movedNodes {
		for _, dependee := range movedNodes {
			if depender == dependee {
				continue
			}
			if policy.DependsOn(depender.MoveOrderingStatement(), dependee.MoveOrderingStatement()) {
				g.Connect(dag.BasicEdge(depender, dependee))
			}
		}
	}

	return nil
}

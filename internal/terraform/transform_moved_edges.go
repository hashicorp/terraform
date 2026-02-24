// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/refactoring"
)

// MovedBlockEdgeTransformer adds ordering edges between moved statements.
type MovedBlockEdgeTransformer struct{}

func (t *MovedBlockEdgeTransformer) Transform(g *Graph) error {
	var movedNodes []*nodeExpandMoved
	for _, v := range g.Vertices() {
		if node, ok := v.(*nodeExpandMoved); ok {
			movedNodes = append(movedNodes, node)
		}
	}

	for _, depender := range movedNodes {
		for _, dependee := range movedNodes {
			if depender == dependee {
				continue
			}
			if refactoring.StatementDependsOn(depender.Stmt, dependee.Stmt) {
				g.Connect(dag.BasicEdge(depender, dependee))
			}
		}
	}

	return nil
}

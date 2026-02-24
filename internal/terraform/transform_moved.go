// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/refactoring"
)

// MovedBlockTransformer adds graph nodes for pre-plan moved block handling.
type MovedBlockTransformer struct {
	Statements []refactoring.MoveStatement
	Runtime    *movedAnalysisRuntime
}

func (t *MovedBlockTransformer) Transform(g *Graph) error {
	for i := range t.Statements {
		g.Add(&nodeExpandMoved{
			Stmt:    &t.Statements[i],
			Index:   i,
			Runtime: t.Runtime,
		})
	}
	return nil
}

// MovedExecutionBlockTransformer adds executable moved nodes using a frozen
// execution order produced by the analysis phase.
type MovedExecutionBlockTransformer struct {
	OrderedStatements []refactoring.MoveStatement
	Runtime           *movedExecutionRuntime
}

func (t *MovedExecutionBlockTransformer) Transform(g *Graph) error {
	var prev dag.Vertex
	for i := range t.OrderedStatements {
		stmt := t.OrderedStatements[i]
		node := &nodeMovedInstance{
			Stmt:    &stmt,
			Index:   i,
			Runtime: t.Runtime,
		}
		g.Add(node)
		if prev != nil {
			// Edges point from depender to dependee. For a frozen execution
			// order [a,b,c], connect b->a and c->b so ReverseTopologicalOrder
			// executes a, then b, then c.
			g.Connect(dag.BasicEdge(node, prev))
		}
		prev = node
	}
	return nil
}

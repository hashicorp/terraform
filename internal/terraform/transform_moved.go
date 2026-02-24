// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import "github.com/hashicorp/terraform/internal/refactoring"

// MovedBlockTransformer adds graph nodes for pre-plan moved block handling.
type MovedBlockTransformer struct {
	Statements []refactoring.MoveStatement
	Runtime    *movedExecutionRuntime
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

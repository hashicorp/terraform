// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/refactoring"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// MovedGraphBuilder constructs a mini-graph that applies move statements
// before the main plan graph walk.
type MovedGraphBuilder struct {
	Statements []refactoring.MoveStatement
	Runtime    *movedExecutionRuntime
}

func (b *MovedGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	return (&BasicGraphBuilder{
		Name:                "MovedGraphBuilder",
		Steps:               b.Steps(),
		SkipGraphValidation: true,
	}).Build(path)
}

func (b *MovedGraphBuilder) Steps() []GraphTransformer {
	return []GraphTransformer{
		&MovedBlockTransformer{
			Statements: b.Statements,
			Runtime:    b.Runtime,
		},
		&MovedBlockEdgeTransformer{},
		&CloseRootModuleTransformer{},
		&TransitiveReductionTransformer{},
	}
}

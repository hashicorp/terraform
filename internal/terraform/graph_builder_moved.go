// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/refactoring"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// MovedGraphBuilder constructs a mini-graph that applies move statements
// before the main plan graph walk.
type MovedGraphBuilder struct {
	Statements []refactoring.MoveStatement
	Runtime    *movedExecutionRuntime

	// Step-2 scaffolding: when enabled, include the minimal eval graph pieces
	// needed for moved-block expression references (for_each) to resolve.
	EnableExpressionEval bool
	Config               *configs.Config
	RootVariableValues   InputValues
}

func (b *MovedGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	return (&BasicGraphBuilder{
		Name:                "MovedGraphBuilder",
		Steps:               b.Steps(),
		SkipGraphValidation: true,
	}).Build(path)
}

func (b *MovedGraphBuilder) Steps() []GraphTransformer {
	var steps []GraphTransformer

	if b.EnableExpressionEval {
		steps = append(steps,
			&RootVariableTransformer{
				Config:    b.Config,
				RawValues: b.RootVariableValues,
				Planning:  true,
			},
			&ModuleVariableTransformer{
				Config:   b.Config,
				Planning: true,
			},
			&LocalTransformer{Config: b.Config},
		)
	}

	steps = append(steps,
		&MovedBlockTransformer{
			Statements: b.Statements,
			Runtime:    b.Runtime,
		},
	)

	if b.EnableExpressionEval {
		steps = append(steps,
			&ModuleExpansionTransformer{Config: b.Config},
			&ReferenceTransformer{},
		)
	}

	steps = append(steps,
		&MovedBlockEdgeTransformer{},
		&CloseRootModuleTransformer{},
		&TransitiveReductionTransformer{},
	)

	return steps
}

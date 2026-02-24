// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/refactoring"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// MovedAnalysisGraphBuilder constructs the move analysis mini-graph, which is
// responsible for expanding moved statement templates into concrete statements
// before execution.
type MovedAnalysisGraphBuilder struct {
	Statements        []refactoring.MoveStatement
	Runtime           *movedAnalysisRuntime
	Config            *configs.Config
	RootVariableValues InputValues
}

func (b *MovedAnalysisGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	return (&BasicGraphBuilder{
		Name:                "MovedAnalysisGraphBuilder",
		Steps:               b.Steps(),
		SkipGraphValidation: true,
	}).Build(path)
}

func (b *MovedAnalysisGraphBuilder) Steps() []GraphTransformer {
	return []GraphTransformer{
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
		&MovedBlockTransformer{
			Statements: b.Statements,
			Runtime:    b.Runtime,
		},
		&ModuleExpansionTransformer{Config: b.Config},
		&ReferenceTransformer{},
		&MovedBlockEdgeTransformer{},
		&CloseRootModuleTransformer{},
		&TransitiveReductionTransformer{},
	}
}

// MovedExecutionGraphBuilder constructs the move execution mini-graph from a
// frozen execution order produced by the analysis phase.
type MovedExecutionGraphBuilder struct {
	OrderedStatements []refactoring.MoveStatement
	Runtime           *movedExecutionRuntime
}

func (b *MovedExecutionGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	return (&BasicGraphBuilder{
		Name:                "MovedExecutionGraphBuilder",
		Steps:               b.Steps(),
		SkipGraphValidation: true,
	}).Build(path)
}

func (b *MovedExecutionGraphBuilder) Steps() []GraphTransformer {
	return []GraphTransformer{
		&MovedExecutionBlockTransformer{
			OrderedStatements: b.OrderedStatements,
			Runtime:           b.Runtime,
		},
		&CloseRootModuleTransformer{},
		&TransitiveReductionTransformer{},
	}
}


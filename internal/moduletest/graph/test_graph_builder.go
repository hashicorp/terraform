// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestGraphBuilder is a GraphBuilder implementation that builds a graph for
// a terraform test file. The file may contain multiple runs, and each run may have
// dependencies on other runs.
type TestGraphBuilder struct {
	File           *moduletest.File
	GlobalVars     map[string]backendrun.UnparsedVariableValue
	ContextOpts    *terraform.ContextOpts
	BackendFactory func(string) backend.InitFn
	StateManifest  *TestManifest
	CommandMode    moduletest.CommandMode
}

type graphOptions struct {
	File          *moduletest.File
	GlobalVars    map[string]backendrun.UnparsedVariableValue
	ContextOpts   *terraform.ContextOpts
	StateManifest *TestManifest
	CommandMode   moduletest.CommandMode
	EvalContext   *EvalContext
}

// See GraphBuilder
func (b *TestGraphBuilder) Build(ctx *EvalContext) (*terraform.Graph, tfdiags.Diagnostics) {
	log.Printf("[TRACE] building graph for terraform test")
	opts := &graphOptions{
		File:          b.File,
		GlobalVars:    b.GlobalVars,
		ContextOpts:   b.ContextOpts,
		StateManifest: b.StateManifest,
		CommandMode:   b.CommandMode,
		EvalContext:   ctx,
	}
	return (&terraform.BasicGraphBuilder{
		Steps: b.Steps(opts),
		Name:  "TestGraphBuilder",
	}).Build(addrs.RootModuleInstance)
}

// See GraphBuilder
func (b *TestGraphBuilder) Steps(opts *graphOptions) []terraform.GraphTransformer {
	steps := []terraform.GraphTransformer{
		&TestRunTransformer{opts},
		&TestStateTransformer{graphOptions: opts, BackendFactory: b.BackendFactory},
		&TestStateCleanupTransformer{opts},
		terraform.DynamicTransformer(validateRunConfigs),
		&TestProvidersTransformer{},
		&CloseTestGraphTransformer{},
		&terraform.TransitiveReductionTransformer{},
	}

	return steps
}

func validateRunConfigs(g *terraform.Graph) error {
	for node := range dag.SelectSeq(g.VerticesSeq(), runFilter) {
		diags := node.run.Config.Validate(node.run.ModuleConfig)
		node.run.Diagnostics = node.run.Diagnostics.Append(diags)
		if diags.HasErrors() {
			node.run.Status = moduletest.Error
		}
	}
	return nil
}

// dynamicNode is a helper node which can be added to the graph to execute
// a dynamic function at some desired point in the graph.
type dynamicNode struct {
	eval func(*EvalContext) tfdiags.Diagnostics
}

func (n *dynamicNode) Execute(evalCtx *EvalContext) tfdiags.Diagnostics {
	return n.eval(evalCtx)
}

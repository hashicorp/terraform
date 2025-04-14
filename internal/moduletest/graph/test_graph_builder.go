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

type CommandMode int

const (
	// NormalMode is the default mode for running terraform test.
	NormalMode CommandMode = iota
	// CleanupMode is used when running terraform test cleanup.
	// In this mode, the graph will be built with the intention of cleaning up
	// the state, rather than applying changes.
	CleanupMode
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
	CommandMode    CommandMode
}

type graphOptions struct {
	File          *moduletest.File
	GlobalVars    map[string]backendrun.UnparsedVariableValue
	ContextOpts   *terraform.ContextOpts
	StateManifest *TestManifest
	CommandMode   CommandMode
}

// See GraphBuilder
func (b *TestGraphBuilder) Build() (*terraform.Graph, tfdiags.Diagnostics) {
	log.Printf("[TRACE] building graph for terraform test")
	return (&terraform.BasicGraphBuilder{
		Steps: b.Steps(),
		Name:  "TestGraphBuilder",
	}).Build(addrs.RootModuleInstance)
}

// See GraphBuilder
func (b *TestGraphBuilder) Steps() []terraform.GraphTransformer {
	opts := &graphOptions{
		File:          b.File,
		GlobalVars:    b.GlobalVars,
		ContextOpts:   b.ContextOpts,
		StateManifest: b.StateManifest,
		CommandMode:   b.CommandMode,
	}
	steps := []terraform.GraphTransformer{
		&TestRunTransformer{opts},
		&TestStateTransformer{graphOptions: opts, BackendFactory: b.BackendFactory},
		&TestStateCleanupTransformer{opts},
		terraform.DynamicTransformer(validateRunConfigs),
		&TestProvidersTransformer{},
		terraform.DynamicTransformer(func(g *terraform.Graph) error {
			// If we're in cleanup mode, we can remove the test runs in the graph,
			// and prevent unnecessary no-op execution.
			// This will ensure that we only have nodes that are needed for cleanup in the graph.
			if b.CommandMode == CleanupMode {
				for node := range dag.SelectSeq(g.VerticesSeq(), runFilter) {
					g.Remove(node)
				}
			}
			return nil
		}),
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

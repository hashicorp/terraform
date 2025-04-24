// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"slices"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

var _ GraphNodeExecutable = &CleanupSubGraph{}

// CleanupSubGraph is a subgraph that is responsible for cleaning up the state of
// resources defined in the state files created by the test runs.
type CleanupSubGraph struct {
	opts *graphOptions
}

func (b *CleanupSubGraph) Execute(ctx *EvalContext) tfdiags.Diagnostics {
	ctx.Renderer().File(b.opts.File, moduletest.TearDown)

	// In cleanup mode, the test run blocks are not executed.
	// Instead, the state file associated with each run is revisited to extract
	// its output values. These output values are then set in the context so
	// that they can be accessed by subsequent cleanup nodes. This is necessary
	// because cleaning up the state file for a run may depend on the output
	// values of previous runs.
	if b.opts.CommandMode == moduletest.CleanupMode {
		for _, run := range b.opts.File.Runs {
			state := ctx.GetFileState(run.Config.StateKey).State
			if state == nil {
				return nil
			}
			outputVals := make(map[string]cty.Value, len(state.RootOutputValues))
			for name, out := range state.RootOutputValues {
				outputVals[name] = out.Value
			}
			ctx.SetOutput(run, cty.ObjectVal(outputVals))
		}
	}

	// Create a new graph for the cleanup nodes
	g, diags := (&terraform.BasicGraphBuilder{
		Steps: []terraform.GraphTransformer{
			&TestStateCleanupTransformer{opts: b.opts},
			&CloseTestGraphTransformer{},
		},
		Name: "TestCleanupSubGraph",
	}).Build(addrs.RootModuleInstance)

	if diags.HasErrors() {
		return diags
	}

	return Walk(g, ctx)
}

// TestStateCleanupTransformer is a GraphTransformer that adds a cleanup node
// for each state that is created by the test runs.
type TestStateCleanupTransformer struct {
	opts *graphOptions
}

func (t *TestStateCleanupTransformer) Transform(g *terraform.Graph) error {
	cleanupMap := make(map[string]*NodeStateCleanup)
	overrideMap := make(map[string]*moduletest.Run)
	for _, run := range t.opts.File.Runs {
		key := run.Config.StateKey

		// If a run is marked as skip_cleanup, that run's apply
		// will be the final state in the state file.
		// This is only relevant to the default test mode.
		if run.Config.SkipCleanup && t.opts.CommandMode != moduletest.CleanupMode {
			overrideMap[key] = run
		}

		// Create a cleanup node for each state key
		if _, exists := cleanupMap[key]; !exists {
			cleanupMap[key] = &NodeStateCleanup{stateKey: key, opts: t.opts}
			g.Add(cleanupMap[key])
		}
	}

	added := make(map[string]bool)
	var prev dag.Vertex

	// Process skip_cleanup attributes and connect all cleanup nodes in
	// reverse order of run index to preserve existing behavior.
	// TODO: Parallelize cleanup nodes execution instead of sequential.
	for _, run := range slices.Backward(t.opts.File.Runs) {
		key := run.Config.StateKey
		node := cleanupMap[key]

		if _, exists := added[key]; !exists {
			if prev != nil {
				g.Connect(dag.BasicEdge(node, prev))
			}
			prev = node
			added[key] = true
			node.customCleanupRun = overrideMap[key]
		}
	}
	return nil
}

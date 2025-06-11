// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"slices"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
)

var _ GraphNodeExecutable = &TeardownSubgraph{}

// TeardownSubgraph is a subgraph for cleaning up the state of
// resources defined in the state files created by the test runs.
type TeardownSubgraph struct {
	opts *graphOptions
}

func (b *TeardownSubgraph) Execute(ctx *EvalContext) {
	ctx.Renderer().File(b.opts.File, moduletest.TearDown)

	// Create a new graph for the cleanup nodes
	g, diags := (&terraform.BasicGraphBuilder{
		Steps: []terraform.GraphTransformer{
			&TestStateCleanupTransformer{opts: b.opts},
			&ReferenceTransformer{},
			&CloseTestGraphTransformer{},
			&terraform.TransitiveReductionTransformer{},
		},
		Name: "TeardownSubgraph",
	}).Build(addrs.RootModuleInstance)
	b.opts.File.Diagnostics = b.opts.File.Diagnostics.Append(diags)

	if diags.HasErrors() {
		return
	}

	diags = Walk(g, ctx)
	b.opts.File.Diagnostics = b.opts.File.Diagnostics.Append(diags)
}

// TestStateCleanupTransformer is a GraphTransformer that adds a cleanup node
// for each state that is created by the test runs.
type TestStateCleanupTransformer struct {
	opts *graphOptions
}

func (t *TestStateCleanupTransformer) Transform(g *terraform.Graph) error {
	cleanupMap := make(map[string]*NodeStateCleanup)
	arr := make([]*NodeStateCleanup, 0, len(t.opts.File.Runs))

	// Sort in reverse order of the run index, so that the last run for each state key
	// is attached to the cleanup node.
	for _, run := range slices.Backward(t.opts.File.Runs) {
		key := run.GetStateKey()
		if _, exists := cleanupMap[key]; !exists {
			refs, _ := run.GetReferences()
			cleanupMap[key] = &NodeStateCleanup{
				stateKey:   key,
				opts:       t.opts,
				parallel:   run.Config.Parallel,
				references: refs,
				addr:       run.Addr(),
			}
			arr = append(arr, cleanupMap[key])
			g.Add(cleanupMap[key])
		}

		// if one of the runs for this state key is not parallel, then
		// the cleanup node should not be parallel either.
		cleanupMap[key].parallel = cleanupMap[key].parallel && run.Config.Parallel
	}

	t.controlParallelism(g, arr)
	return nil
}

func (t *TestStateCleanupTransformer) controlParallelism(g *terraform.Graph, nodes []*NodeStateCleanup) {
	// If there is a state that has opted out of parallelism, we will connect it
	// sequentially to all previous and subsequent runs.
	for i, node := range nodes {
		if node.parallel {
			continue
		}

		// Connect to all previous runs
		for j := 0; j < i; j++ {
			g.Connect(dag.BasicEdge(node, nodes[j]))
		}

		// Connect to all subsequent runs
		for j := i + 1; j < len(nodes); j++ {
			g.Connect(dag.BasicEdge(nodes[j], node))
		}
	}
}

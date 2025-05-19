// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"slices"

	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestStateCleanupTransformer is a GraphTransformer that adds a cleanup node
// for each state that is created by the test runs.
type TestStateCleanupTransformer struct {
	opts *graphOptions
}

func (t *TestStateCleanupTransformer) Transform(g *terraform.Graph) error {
	cleanupMap := make(map[string]*NodeStateCleanup)

	for _, v := range g.Vertices() {
		node, ok := v.(*NodeTestRun)
		if !ok {
			continue
		}
		key := node.run.GetStateKey()
		if _, exists := cleanupMap[key]; !exists {
			cleanupMap[key] = &NodeStateCleanup{stateKey: key, opts: t.opts}
			g.Add(cleanupMap[key])
		}

		// Connect the cleanup node to the test run node.
		g.Connect(dag.BasicEdge(cleanupMap[key], node))
	}

	// Add a root cleanup node that runs before cleanup nodes for each state.
	// Right now it just simply renders a teardown summary, so as to maintain
	// existing CLI output.
	rootCleanupNode := t.addRootCleanupNode(g)

	for _, v := range g.Vertices() {
		switch node := v.(type) {
		case *NodeTestRun:
			// All the runs that share the same state, must share the same cleanup node,
			// which only executes once after all the dependent runs have completed.
			g.Connect(dag.BasicEdge(rootCleanupNode, node))
		case *NodeStateCleanup:
			// Connect the cleanup node to the root cleanup node.
			g.Connect(dag.BasicEdge(node, rootCleanupNode))
		}
	}

	// connect all cleanup nodes in reverse-sequential order of run index to
	// preserve existing behavior, starting from the root cleanup node.
	// TODO: Parallelize cleanup nodes execution instead of sequential.
	added := make(map[string]bool)
	var prev dag.Vertex
	for _, v := range slices.Backward(t.opts.File.Runs) {
		key := v.GetStateKey()
		if _, exists := added[key]; !exists {
			node := cleanupMap[key]
			if prev != nil {
				g.Connect(dag.BasicEdge(node, prev))
			}
			prev = node
			added[key] = true
		}
	}

	return nil
}

func (t *TestStateCleanupTransformer) addRootCleanupNode(g *terraform.Graph) *dynamicNode {
	rootCleanupNode := &dynamicNode{
		eval: func(ctx *EvalContext) tfdiags.Diagnostics {
			var diags tfdiags.Diagnostics
			ctx.Renderer().File(t.opts.File, moduletest.TearDown)
			return diags
		},
	}
	g.Add(rootCleanupNode)
	return rootCleanupNode
}

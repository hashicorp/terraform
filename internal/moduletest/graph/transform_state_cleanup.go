// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"sort"

	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
)

// TestStateCleanupTransformer is a GraphTransformer that adds a cleanup node
// for each state that is created by the test runs.
type TestStateCleanupTransformer struct {
	opts *graphOptions
}

func (t *TestStateCleanupTransformer) Transform(g *terraform.Graph) error {
	cleanupMap := make(map[string]*NodeStateCleanup)
	arr := make([]*NodeStateCleanup, 0, len(t.opts.File.Runs))

	for node := range dag.SelectSeq[*NodeTestRun](g.VerticesSeq()) {
		key := node.run.GetStateKey()
		if _, exists := cleanupMap[key]; !exists {
			cleanupMap[key] = &NodeStateCleanup{stateKey: key, opts: t.opts, parallel: node.run.Config.Parallel}
			arr = append(arr, cleanupMap[key])
			g.Add(cleanupMap[key])
		}

		// if one of the runs for this state key is not parallel, then
		// the cleanup node should not be parallel either.
		cleanupMap[key].parallel = cleanupMap[key].parallel && node.run.Config.Parallel
		cleanupMap[key].run = node

		// Connect the cleanup node to the test run node.
		g.Connect(dag.BasicEdge(cleanupMap[key], node))
	}

	// Add a root cleanup node that runs before cleanup nodes for each state.
	// Right now it just simply renders a teardown summary, so as to maintain
	// existing CLI output.
	rootCleanupNode := t.addRootCleanupNode(g)

	for v := range g.VerticesSeq() {
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

	t.controlParallelism(g, arr)
	return nil
}

func (t *TestStateCleanupTransformer) controlParallelism(g *terraform.Graph, nodes []*NodeStateCleanup) {
	// Sort in reverse order of the run index
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].run == nil || nodes[j].run == nil {
			return false
		}
		return nodes[i].run.run.Index > nodes[j].run.run.Index
	})

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

func (t *TestStateCleanupTransformer) addRootCleanupNode(g *terraform.Graph) *dynamicNode {
	rootCleanupNode := &dynamicNode{
		eval: func(ctx *EvalContext) {
			ctx.Renderer().File(t.opts.File, moduletest.TearDown)
		},
	}
	g.Add(rootCleanupNode)
	return rootCleanupNode
}

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
	opts   *graphOptions
	parent *terraform.Graph
}

func (b *TeardownSubgraph) Execute(ctx *EvalContext) {
	ctx.Renderer().File(b.opts.File, moduletest.TearDown)

	// work out the transitive run references for each run node in the parent graph
	runRefMap := make(map[addrs.Run][]string)
	for runNode := range dag.SelectSeq[*NodeTestRun](b.parent.VerticesSeq()) {
		refs := b.parent.Ancestors(runNode)
		for _, ref := range refs {
			if ref, ok := ref.(*NodeTestRun); ok {
				if ref.run.GetStateKey() != runNode.run.GetStateKey() {
					runRefMap[runNode.run.Addr()] = append(runRefMap[runNode.run.Addr()], ref.run.GetStateKey())
				}
			}
		}
	}

	// Create a new graph for the cleanup nodes
	g, diags := (&terraform.BasicGraphBuilder{
		Steps: []terraform.GraphTransformer{
			&TestVariablesTransformer{File: b.opts.File},
			&TestStateCleanupTransformer{opts: b.opts, runRefs: runRefMap},
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
	opts    *graphOptions
	runRefs map[addrs.Run][]string
}

func (t *TestStateCleanupTransformer) Transform(g *terraform.Graph) error {
	cleanupMap := make(map[string]*NodeStateCleanup)
	arr := make([]*NodeStateCleanup, 0, len(t.opts.File.Runs))

	// dependency map for state keys, which will be used to traverse
	// the cleanup nodes in a depth-first manner.
	depStateKeys := make(map[string][]string)

	// Sort in reverse order of the run index, so that the last run for each state key
	// is attached to the cleanup node.
	for _, run := range slices.Backward(t.opts.File.Runs) {
		key := run.GetStateKey()

		if _, exists := cleanupMap[key]; !exists {
			node := &NodeStateCleanup{
				stateKey: key,
				opts:     t.opts,
				parallel: run.Config.Parallel,
			}
			cleanupMap[key] = node
			arr = append(arr, node)
			g.Add(node)

			// Build the dependency map for this state key.
			refStateKeys := t.runRefs[run.Addr()]
			depStateKeys[key] = make([]string, 0, len(refStateKeys))
			for _, refKey := range refStateKeys {
				depStateKeys[key] = append(depStateKeys[key], refKey)
			}
			continue
		}

		// if one of the runs for this state key is not parallel, then
		// the cleanup node should not be parallel either.
		cleanupMap[key].parallel = cleanupMap[key].parallel && run.Config.Parallel
	}

	// Depth-first traversal to connect the cleanup nodes based on their dependencies.
	// If an edge would create a cycle, we skip it.
	visited := make(map[string]bool)
	for _, node := range arr {
		t.depthFirstTraverse(g, node, visited, cleanupMap, depStateKeys)
	}

	t.controlParallelism(g, arr)
	return nil
}

func (t *TestStateCleanupTransformer) depthFirstTraverse(g *terraform.Graph, node *NodeStateCleanup, visited map[string]bool, cleanupNodes map[string]*NodeStateCleanup, depStateKeys map[string][]string) {
	if node == nil || visited[node.stateKey] {
		return
	}
	visited[node.stateKey] = true

	for _, refStateKey := range depStateKeys[node.stateKey] {
		refNode, exists := cleanupNodes[refStateKey]
		if !exists || visited[refNode.stateKey] {
			continue
		}
		g.Connect(dag.BasicEdge(refNode, node))
		t.depthFirstTraverse(g, refNode, visited, cleanupNodes, depStateKeys)
	}
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

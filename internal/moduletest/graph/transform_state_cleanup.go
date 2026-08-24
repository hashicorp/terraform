// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"slices"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
)

var (
	_ GraphNodeExecutable = &TeardownSubgraph{}
	_ Subgrapher          = &TeardownSubgraph{}
)

type Subgrapher interface {
	isSubGrapher()
}

// RunNode is an interface for nodes that represent test runs.
// As of now, there are mainly two types of run nodes:
// - NodeTestRun: represents a test run that is executed as part of a module test.
// - NodeTestRunCleanup: represents a node for cleaning up a test run's state.
type RunNode interface {
	dag.Vertex
	Run() *moduletest.Run
}

// TeardownSubgraph is a subgraph for cleaning up the state of
// resources defined in the state files created by test runs.
type TeardownSubgraph struct {
	opts *graphOptions

	// runGraph is the execution graph containing the test runs.
	runGraph *terraform.Graph
	mode     moduletest.CommandMode
}

func (g *TeardownSubgraph) Name() string {
	return "TeardownSubgraph"
}

func (b *TeardownSubgraph) Execute(ctx *EvalContext) {
	ctx.Renderer().File(b.opts.File, moduletest.TearDown)

	stateDepMap := make(map[addrs.Run][]string)

	// Build a map of run nodes to other run nodes they depend on.
	// In cleanup mode, the run node is the NodeTestRunCleanup struct.
	for runNode := range dag.SelectSeq[RunNode](b.runGraph.VerticesSeq()) {
		addr := runNode.Run().Addr()
		refs := b.runGraph.Ancestors(runNode)
		for ref := range refs.All() {
			if ref, ok := ref.(RunNode); ok && ref.Run().Config.StateKey != runNode.Run().Config.StateKey {
				stateDepMap[addr] = append(stateDepMap[addr], ref.Run().Config.StateKey)
			}
		}
	}

	// Create a new graph for the cleanup nodes
	g, diags := (&terraform.BasicGraphBuilder{
		Steps: []terraform.GraphTransformer{
			&TestStateCleanupTransformer{opts: b.opts, stateDependencyMap: stateDepMap},
			&CloseTestGraphTransformer{},
			&terraform.TransitiveReductionTransformer{},
		},
		Name: "TeardownSubgraph",
	}).Build(addrs.RootModuleInstance)
	b.opts.File.AppendDiagnostics(diags)

	if diags.HasErrors() {
		return
	}

	diags = Walk(g, ctx)
	b.opts.File.AppendDiagnostics(diags)
}

func (b *TeardownSubgraph) isSubGrapher() {}

// TestStateCleanupTransformer is a GraphTransformer that adds a cleanup node
// for each state that is created by the test runs.
type TestStateCleanupTransformer struct {
	opts               *graphOptions
	stateDependencyMap map[addrs.Run][]string
}

func (t *TestStateCleanupTransformer) Transform(g *terraform.Graph) error {
	cleanupMap := make(map[string]*NodeStateCleanup)
	arr := make([]*NodeStateCleanup, 0, len(t.opts.File.Runs))

	// dependency map for state keys, which will be used to traverse
	// the cleanup nodes in a depth-first manner.
	depStateKeys := make(map[string][]string)

	// iterate in reverse order of the run index, so that the last run for each state key
	// is attached to the cleanup node.
	for _, run := range slices.Backward(t.opts.File.Runs) {
		key := run.Config.StateKey

		if _, exists := cleanupMap[key]; !exists {
			node := &NodeStateCleanup{
				stateKey: key,
				opts:     t.opts,
			}
			cleanupMap[key] = node
			arr = append(arr, node)
			g.Add(node)

			// The dependency map for the state's last run will be used for the cleanup node.
			depStateKeys[key] = t.stateDependencyMap[run.Addr()]
		}
	}

	// Depth-first traversal to connect the cleanup nodes based on their dependencies.
	// If an edge would create a cycle, we skip it.
	visited := make(map[string]bool)
	for _, node := range arr {
		t.depthFirstTraverse(g, node, visited, cleanupMap, depStateKeys)
	}
	return nil
}

func (t *TestStateCleanupTransformer) depthFirstTraverse(g *terraform.Graph, node *NodeStateCleanup, visited map[string]bool, cleanupNodes map[string]*NodeStateCleanup, depStateKeys map[string][]string) {
	if visited[node.stateKey] {
		return
	}
	// don't mark the node as visited if it's a leaf node, this ensures that other dependencies are still added to it
	if len(depStateKeys[node.stateKey]) == 0 {
		return
	}
	visited[node.stateKey] = true

	for _, refStateKey := range depStateKeys[node.stateKey] {
		// If the reference node has already been visited, skip it.
		if visited[refStateKey] {
			continue
		}
		refNode := cleanupNodes[refStateKey]
		g.Connect(refNode, node)
		t.depthFirstTraverse(g, refNode, visited, cleanupNodes, depStateKeys)
	}
}

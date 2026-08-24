// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"maps"
	"slices"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
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

	stateDepMap := make(map[addrs.Run]collections.Set[string])

	// Build a map of run nodes to other run nodes they depend on.
	// In cleanup mode, the run node is the NodeTestRunCleanup struct.
	for runNode := range dag.SelectSeq[RunNode](b.runGraph.VerticesSeq()) {
		addr := runNode.Run().Addr()
		stateDepMap[addr] = collections.NewSetCmp[string]()
		refs := b.runGraph.Ancestors(runNode)
		for ref := range refs.All() {
			if ref, ok := ref.(RunNode); ok && ref.Run().Config.StateKey != runNode.Run().Config.StateKey {
				stateDepMap[addr].Add(ref.Run().Config.StateKey)
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
	stateDependencyMap map[addrs.Run]collections.Set[string]
}

func (t *TestStateCleanupTransformer) Transform(g *terraform.Graph) error {
	cleanupMap := make(map[string]*NodeStateCleanup)
	cleanupNodes := make([]*NodeStateCleanup, 0, len(t.opts.File.Runs))

	// dependency map for state keys, which will be used to traverse
	// the cleanup nodes in a depth-first manner.
	depStateKeys := make(map[string]collections.Set[string])
	collections.NewSetCmp[string]()

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
			cleanupNodes = append(cleanupNodes, node)
			g.Add(node)

			// The dependency map for the state's last run will be used for the cleanup node.
			depStateKeys[key] = t.stateDependencyMap[run.Addr()]
		}
	}

	// Depth-first traversal to connect the cleanup nodes based on their dependencies.
	// This traversal ensures that nodes are only visited once.
	// Each visited node processes its dependencies and connects cleanup nodes as needed, so if
	// another path leads to the same node, it will not be processed again.
	visited := make(map[string]bool)
	for _, node := range cleanupNodes {
		if visited[node.stateKey] {
			continue
		}

		// The processed map tracks nodes encountered along this node's path.
		processed := make(map[string]bool)
		t.depthFirstTraverse(g, node, processed, cleanupMap, depStateKeys)

		// Mark the node and its dependencies as visited to avoid processing them again.
		maps.Copy(visited, processed)
	}
	if cy := g.Cycles(); len(cy) > 0 {
		return fmt.Errorf("cleanup graph contains cycles")
	}
	return nil
}

// depthFirstTraverse is a recursive helper function that traverses the cleanup graph depth-first.
// The cleanup graph must preserve reverse run order, but references between runs can introduce edges that
// conflict with this order, especially when multiple runs use the same state key. When an edge would introduce
// a cycle, it is skipped to avoid invalidating the graph.
//
// For example: Given the order
// test_one (S) -> test_two (T) -> test_three (S), where S and T are state keys
// In this case, the cleanup graph must be "S -> T".
// If test_two were to reference test_one, an edge like "T -> S" is requested, but that would introduce a cycle.
// Therefore, the edge is skipped, and no cycle is introduced.
func (t *TestStateCleanupTransformer) depthFirstTraverse(g *terraform.Graph, node *NodeStateCleanup, visited map[string]bool, cleanupNodes map[string]*NodeStateCleanup, depStateKeys map[string]collections.Set[string]) {
	visited[node.stateKey] = true

	for depStateKey := range depStateKeys[node.stateKey].All() {
		// If the reference node has already been visited along the current path, skip it.
		if visited[depStateKey] {
			continue
		}
		refNode := cleanupNodes[depStateKey]
		g.Connect(refNode, node)
		t.depthFirstTraverse(g, refNode, visited, cleanupNodes, depStateKeys)
	}
}

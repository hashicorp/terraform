// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
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

	// Create a new graph for the cleanup nodes
	g, diags := (&terraform.BasicGraphBuilder{
		Steps: []terraform.GraphTransformer{
			&TestStateCleanupTransformer{opts: b.opts, runGraph: b.runGraph},
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
	opts     *graphOptions
	runGraph *terraform.Graph
}

func (t *TestStateCleanupTransformer) Transform(g *terraform.Graph) error {
	cleanupRuns := collections.NewMapCmp[string, string]()
	cleanupNodes := collections.NewMapCmp[string, *NodeStateCleanup]()

	// iterate in topological order of the run graph, so that the most re run for each state key
	// is attached to the cleanup node.
	for _, node := range t.runGraph.TopologicalOrder() {
		runNode, ok := node.(RunNode)
		if !ok {
			continue
		}
		run := runNode.Run()
		key := run.Config.StateKey

		if _, ok := cleanupRuns.GetOk(key); !ok {
			cleanupRuns.Put(key, runNode.Run().Name)
			cleanupNode := &NodeStateCleanup{stateKey: key, opts: t.opts}
			g.Add(cleanupNode)
			cleanupNodes.Put(key, cleanupNode)
		}
	}

	isCleanupRun := func(v dag.Vertex) (RunNode, bool) {
		node, ok := v.(RunNode)
		if !ok {
			return nil, false
		}
		nodeName, ok := cleanupRuns.GetOk(node.Run().Config.StateKey)
		if !ok {
			return nil, false
		}

		// return true if the node is the last run with the given state key
		return node, nodeName == node.Run().Name
	}

	// Traverse the run graph in topological order and build the cleanup edges
	// in reverse order of the run graph.
	for _, node := range t.runGraph.TopologicalOrder() {
		sourceNode, ok := isCleanupRun(node)
		if !ok {
			continue
		}
		sourceKey := sourceNode.Run().Config.StateKey
		cleanupNode := cleanupNodes.Get(sourceKey)

		// For each cleanup node, connect it to the cleanup nodes found in
		// its transitive dependencies. Intermediate nodes are intentionally
		// omitted from the cleanup graph.
		deps := t.runGraph.Ancestors(node)
		for dep := range deps.All() {
			targetNode, ok := isCleanupRun(dep)
			if !ok {
				continue
			}
			targetKey := targetNode.Run().Config.StateKey

			// The source and one of its dependencies share the same state key,
			// so we do not need to connect them directly.
			if sourceKey == targetKey {
				continue
			}

			depCleanupNode := cleanupNodes.Get(targetKey)
			g.Connect(depCleanupNode, cleanupNode)
		}
	}
	return nil
}

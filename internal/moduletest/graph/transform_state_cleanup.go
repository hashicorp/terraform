// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"

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
	cleanupMap := make(map[string]*NodeStateCleanup)
	cleanupNodes := collections.NewSetCmp[dag.Vertex]()

	// iterate in topological order of the run graph, so that the last run for each state key
	// is attached to the cleanup node.
	for _, node := range t.runGraph.TopologicalOrder() {
		runNode, ok := node.(RunNode)
		if !ok {
			continue
		}
		run := runNode.Run()
		key := run.Config.StateKey

		if _, exists := cleanupMap[key]; !exists {
			node := &NodeStateCleanup{stateKey: key, opts: t.opts}
			cleanupMap[key] = node
			cleanupNodes.Add(runNode)
			g.Add(node)
		}
	}

	// Traverse the run graph in topological order and build the cleanup edges
	// in reverse order of the run graph.
	t.runGraph.TopologicalTraversal(func(v dag.Vertex, dep dag.Vertex) {
		// filter non cleanup nodes
		if !cleanupNodes.Has(v) || !cleanupNodes.Has(dep) {
			return
		}

		// No type assertion needed here, the cleanupNodes set ensures we only process cleanup nodes.
		node := v.(RunNode)
		depNode := dep.(RunNode)

		stateKey := node.Run().Config.StateKey
		depStateKey := depNode.Run().Config.StateKey

		cleanupNode := cleanupMap[stateKey]
		depCleanupNode := cleanupMap[depStateKey]

		// connect the edges in reverse order of the run graph
		g.Connect(cleanupNode, depCleanupNode)
	})
	if err := g.Validate(); err != nil {
		return fmt.Errorf("Invalid cleanup graph: %w", err)
	}
	return nil
}

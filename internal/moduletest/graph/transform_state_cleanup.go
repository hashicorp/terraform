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
	TestRun() *moduletest.Run
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

	// stateOwners maps state keys to the names of the runs that own them.
	stateOwners := make(map[string]string)
	for _, run := range b.opts.File.Runs {
		state := ctx.getState(run.Config.StateKey)
		if state.Run.Name == run.Name {
			stateOwners[run.Config.StateKey] = run.Name
		}
	}

	// Create a new graph for the cleanup nodes
	g, diags := (&terraform.BasicGraphBuilder{
		Steps: []terraform.GraphTransformer{
			&TestStateCleanupTransformer{opts: b.opts, runGraph: b.runGraph, stateOwners: stateOwners},
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
	opts        *graphOptions
	runGraph    *terraform.Graph
	stateOwners map[string]string
}

func (t *TestStateCleanupTransformer) Transform(g *terraform.Graph) error {
	cleanupNodes := collections.NewMapCmp[string, *NodeStateCleanup]()

	// create cleanup nodes for each state key. The state is either owned by
	// the most recent run or the first run with skip_cleanup=true.
	for _, run := range t.opts.File.Runs {
		key := run.Config.StateKey
		runName := t.stateOwners[key]
		if runName != run.Name {
			continue
		}

		cleanupNode := &NodeStateCleanup{stateKey: key, opts: t.opts}
		g.Add(cleanupNode)
		cleanupNodes.Put(key, cleanupNode)
	}

	// helper function to determine if a node owns the state it is associated with.
	stateOwner := func(v dag.Vertex) (*moduletest.Run, bool) {
		node, ok := v.(RunNode)
		if !ok {
			return nil, false
		}

		runName := t.stateOwners[node.TestRun().Config.StateKey]

		// return true if the node owns the state.
		return node.TestRun(), runName == node.TestRun().Name
	}

	// Traverse the run graph and build the cleanup edges
	// in reverse order of the run graph.
	for node := range t.runGraph.VerticesSeq() {
		// is this node a test run and a state owner?
		sourceRun, ok := stateOwner(node)
		if !ok {
			continue
		}
		sourceKey := sourceRun.Config.StateKey
		cleanupNode := cleanupNodes.Get(sourceKey)

		// For each cleanup node, we use its corresponding run node dependencies
		// for ordering. The cleanup nodes are connected to their dependencies
		// in reverse order, i.e each run cleanup node is executed before the run's dependencies.
		dependencies := t.runGraph.Ancestors(node)
		for dependency := range dependencies.All() {
			targetRun, ok := stateOwner(dependency)
			if !ok {
				continue
			}
			targetKey := targetRun.Config.StateKey

			// The source and this dependency share the same state key,
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

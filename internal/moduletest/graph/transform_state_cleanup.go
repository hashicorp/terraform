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

	runRefs := make(map[addrs.Run][]*moduletest.Run)

	// Build a map of run nodes to other run nodes they depend on.
	// In cleanup mode, the run node is the NodeTestRunCleanup struct.
	for runNode := range dag.SelectSeq[RunNode](b.runGraph.VerticesSeq()) {
		addr := runNode.Run().Addr()
		refs := b.runGraph.Ancestors(runNode)
		for ref := range refs.All() {
			if ref, ok := ref.(RunNode); ok {
				runRefs[addr] = append(runRefs[addr], ref.Run())
			}
		}
	}

	// Create a new graph for the cleanup nodes
	g, diags := (&terraform.BasicGraphBuilder{
		Steps: []terraform.GraphTransformer{
			&TestStateCleanupTransformer{opts: b.opts, runDependencyMap: runRefs},
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
	opts             *graphOptions
	runDependencyMap map[addrs.Run][]*moduletest.Run
}

func (t *TestStateCleanupTransformer) Transform(g *terraform.Graph) error {
	type cleanupObj struct {
		node         *NodeStateCleanup
		dependencies []*moduletest.Run
	}

	cleanupMap := make(map[string]cleanupObj)
	runNodesUsedForCleanup := make(map[addrs.Run]bool)

	// iterate in reverse order of the run index, so that the dependency map of the last
	// run for each state key is used for the cleanup node.
	for _, run := range slices.Backward(t.opts.File.Runs) {
		key := run.Config.StateKey

		if _, exists := cleanupMap[key]; !exists {
			node := &NodeStateCleanup{
				stateKey: key,
				opts:     t.opts,
			}
			cleanupMap[key] = cleanupObj{
				node:         node,
				dependencies: t.runDependencyMap[run.Addr()],
			}
			g.Add(node)
			runNodesUsedForCleanup[run.Addr()] = true
			continue
		}
	}

	// We connect the cleanup nodes to their dependencies in reverse order,
	// i.e a cleanup node for a run will evaluate before its references.
	// We only connect references that are also cleanup nodes. If a referenced run
	// is not used by a cleanup node, it will not be connected.
	for _, obj := range cleanupMap {
		for _, dep := range obj.dependencies {
			if _, exists := runNodesUsedForCleanup[dep.Addr()]; exists {
				depCleanupNode := cleanupMap[dep.Config.StateKey].node
				objCleanupNode := obj.node
				if depCleanupNode == objCleanupNode {
					continue
				}
				g.Connect(depCleanupNode, objCleanupNode)
			}
		}
	}
	return nil
}

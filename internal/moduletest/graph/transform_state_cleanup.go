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

// TeardownSubgraph is a subgraph for cleaning up the state of
// resources defined in the state files created by the test runs.
type TeardownSubgraph struct {
	opts     *graphOptions
	runGraph *terraform.Graph
	mode     moduletest.CommandMode
}

func (b *TeardownSubgraph) Execute(ctx *EvalContext) {
	ctx.Renderer().File(b.opts.File, moduletest.TearDown)

	runRefs := make(map[addrs.Run][]*moduletest.Run)

	// Build a map of run nodes to other run nodes they depend on.
	// In cleanup mode, the run node is the NodeTestRunCleanup struct.
	if b.mode == moduletest.CleanupMode {
		for runNode := range dag.SelectSeq[*NodeTestRunCleanup](b.runGraph.VerticesSeq()) {
			addr := runNode.run.Addr()
			parents := b.runGraph.Ancestors(runNode)
			for _, ref := range parents {
				if ref, ok := ref.(*NodeTestRunCleanup); ok {
					runRefs[addr] = append(runRefs[addr], ref.run)
				}
			}
		}
	} else {
		for runNode := range dag.SelectSeq[*NodeTestRun](b.runGraph.VerticesSeq()) {
			addr := runNode.run.Addr()
			parents := b.runGraph.Ancestors(runNode)
			for _, ref := range parents {
				if ref, ok := ref.(*NodeTestRun); ok {
					runRefs[addr] = append(runRefs[addr], ref.run)
				}
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
				g.Connect(dag.BasicEdge(depCleanupNode, objCleanupNode))
			}
		}
	}
	return nil
}

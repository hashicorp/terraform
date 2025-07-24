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
	opts   *graphOptions
	parent *terraform.Graph
}

func (b *TeardownSubgraph) Execute(ctx *EvalContext) {
	ctx.Renderer().File(b.opts.File, moduletest.TearDown)

	runRefMap := make(map[addrs.Run][]string)

	if b.opts.CommandMode == moduletest.CleanupMode {
		for runNode := range dag.SelectSeq[*NodeTestRunCleanup](b.parent.VerticesSeq()) {
			refs := b.parent.Ancestors(runNode)
			for _, ref := range refs {
				if ref, ok := ref.(*NodeTestRunCleanup); ok && ref.run.Config.StateKey != runNode.run.Config.StateKey {
					runRefMap[runNode.run.Addr()] = append(runRefMap[runNode.run.Addr()], ref.run.Config.StateKey)
				}
			}
		}
	} else {
		for runNode := range dag.SelectSeq[*NodeTestRun](b.parent.VerticesSeq()) {
			refs := b.parent.Ancestors(runNode)
			for _, ref := range refs {
				if ref, ok := ref.(*NodeTestRun); ok && ref.run.Config.StateKey != runNode.run.Config.StateKey {
					runRefMap[runNode.run.Addr()] = append(runRefMap[runNode.run.Addr()], ref.run.Config.StateKey)
				}
			}
		}
	}

	// Create a new graph for the cleanup nodes
	g, diags := (&terraform.BasicGraphBuilder{
		Steps: []terraform.GraphTransformer{
			&TestStateCleanupTransformer{opts: b.opts, runStateRefs: runRefMap},
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
	opts         *graphOptions
	runStateRefs map[addrs.Run][]string
}

func (t *TestStateCleanupTransformer) Transform(g *terraform.Graph) error {
	cleanupMap := make(map[string]*NodeStateCleanup)
	overrideMap := make(map[string]*moduletest.Run)
	var arr []*NodeStateCleanup

	for _, run := range t.opts.File.Runs {
		key := run.Config.StateKey

		// If a run is marked as skip_cleanup, that run's apply
		// will be the final state in the state file.
		// This is only relevant to the default test mode.
		if run.Config.SkipCleanup && t.opts.CommandMode != moduletest.CleanupMode {
			overrideMap[key] = run
		}
	}

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
			node.customCleanupRun = overrideMap[key]
			g.Add(node)

			// The dependency map for the state's last run will be used for the cleanup node.
			depStateKeys[key] = t.runStateRefs[run.Addr()]
			continue
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
	visited[node.stateKey] = true

	for _, refStateKey := range depStateKeys[node.stateKey] {
		// If the reference node has already been visited, skip it.
		if visited[refStateKey] {
			continue
		}
		refNode := cleanupNodes[refStateKey]
		g.Connect(dag.BasicEdge(refNode, node))
		t.depthFirstTraverse(g, refNode, visited, cleanupNodes, depStateKeys)
	}
}

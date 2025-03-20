// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"slices"

	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestStateCleanupTransformer is a GraphTransformer that adds a cleanup node
// for each state that is created by the test runs.
type TestStateCleanupTransformer struct {
	opts *graphOptions
}

func (t *TestStateCleanupTransformer) Transform(g *terraform.Graph) error {
	cleanupMap := make(map[string]*NodeStateCleanup)
	runsMap := make(map[string]*NodeTestRun)

	// Add a root cleanup node that runs before cleanup nodes for each state.
	// Right now it just simply renders a teardown summary, so as to maintain
	// existing CLI output.
	rootCleanupNode := t.addRootCleanupNode(g)

	for node := range dag.SelectSeq(g.VerticesSeq(), runFilter) {
		runsMap[node.run.Config.Name] = node // Add all runs to the map, so we can reference them by name later

		// Create a cleanup node for each state key
		key := node.run.Config.StateKey
		if _, exists := cleanupMap[key]; !exists {
			cleanupMap[key] = &NodeStateCleanup{stateKey: key, opts: t.opts}
			g.Add(cleanupMap[key])
		}

		// All the runs that share the same state, must share the same cleanup node,
		// which only executes once after all the dependent runs have completed.
		g.Connect(dag.BasicEdge(cleanupMap[key], node))

		// Connect cleanup nodes to root cleanup node
		g.Connect(dag.BasicEdge(cleanupMap[key], rootCleanupNode))

		// All runs must be executed before the root cleanup node
		g.Connect(dag.BasicEdge(rootCleanupNode, node))

	}

	// Keep track of processed state keys to avoid duplicate connections
	added := make(map[string]bool)
	var prev dag.Vertex

	// Process skip_cleanup attributes and connect all cleanup nodes in
	// reverse order of run index to preserve existing behavior.
	// TODO: Parallelize cleanup nodes execution instead of sequential.
	for _, v := range slices.Backward(t.opts.File.Runs) {
		key := v.Config.StateKey
		node := cleanupMap[key]

		// Process each state key only once
		if _, exists := added[key]; !exists {
			if prev != nil {
				g.Connect(dag.BasicEdge(node, prev))
			}
			prev = node
			added[key] = true
		}

		// Handle skip_cleanup attribute
		if v.Config.SkipCleanup {
			if node.applyOverride != nil { // the node already has an applyOverride from a later run
				v.Diagnostics = v.Diagnostics.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					"Multiple runs with skip_cleanup set",
					fmt.Sprintf(`The run %q has skip_cleanup set to true, but shares state with a later run %q that also has skip_cleanup set. The later run takes precedence, and this attribute is ignored for the earlier run.`,
						v.Config.Name, node.applyOverride.Config.Name),
				))
				continue
			}

			node.applyOverride = v
		}
	}

	return nil
}

func (t *TestStateCleanupTransformer) addRootCleanupNode(g *terraform.Graph) *dynamicNode {
	rootCleanupNode := &dynamicNode{
		eval: func(ctx *EvalContext) tfdiags.Diagnostics {
			var diags tfdiags.Diagnostics
			ctx.Renderer().File(t.opts.File, moduletest.TearDown)
			return diags
		},
	}
	g.Add(rootCleanupNode)
	return rootCleanupNode
}

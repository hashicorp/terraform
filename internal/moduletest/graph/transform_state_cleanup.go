// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"slices"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var _ GraphNodeExecutable = &CleanupSubGraph{}

type CleanupSubGraph struct {
	opts *graphOptions
}

func (b *CleanupSubGraph) Execute(ctx *EvalContext) tfdiags.Diagnostics {
	ctx.Renderer().File(b.opts.File, moduletest.TearDown)
	g, diags := (&terraform.BasicGraphBuilder{
		Steps: []terraform.GraphTransformer{
			&TestStateCleanupTransformer{opts: b.opts},
			&CloseTestGraphTransformer{},
		},
		Name: "TestCleanupGraph",
	}).Build(addrs.RootModuleInstance)

	if diags.HasErrors() {
		return diags
	}

	return Walk(g, ctx, terraform.NewSemaphore(10))
}

// TestStateCleanupTransformer is a GraphTransformer that adds a cleanup node
// for each state that is created by the test runs.
type TestStateCleanupTransformer struct {
	opts *graphOptions
}

func (t *TestStateCleanupTransformer) Transform(g *terraform.Graph) error {
	cleanupMap := make(map[string]*NodeStateCleanup)
	overrideMap := make(map[string]*moduletest.Run)
	for _, run := range t.opts.File.Runs {

		// if skip_cleanup is set, we store the run in the overrideMap
		if run.Config.SkipCleanup {
			overrideMap[run.Config.StateKey] = run
		}

		// Create a cleanup node for each run
		cleanupMap[run.Name] = &NodeStateCleanup{run: run, opts: t.opts, repair: t.opts.EvalContext.repair}
		g.Add(cleanupMap[run.Name])
	}

	// Keep track of processed state keys to avoid duplicate connections
	added := make(map[string]bool)
	var prev dag.Vertex

	// Process skip_cleanup attributes and connect all cleanup nodes in
	// reverse order of run index to preserve existing behavior.
	// TODO: Parallelize cleanup nodes execution instead of sequential.
	for _, v := range slices.Backward(t.opts.File.Runs) {
		key := v.Name
		node := cleanupMap[key]

		if _, exists := added[key]; !exists {
			if prev != nil {
				g.Connect(dag.BasicEdge(node, prev))
				fmt.Printf("%s -> %s\n", node.Name(), prev.(*NodeStateCleanup).Name())
			}
			prev = node
			added[key] = true
		}

		node.applyOverride = overrideMap[v.Config.StateKey]

		// Check if the run has a skip_cleanup attribute set, and set it only once
		// for each state key.
		// override := overrides[v.Config.StateKey]
		// if v.Config.SkipCleanup {
		// 	// the node already has an applyOverride from a later run
		// 	if override != nil && v.Config.SkipCleanupSet {
		// 		// We already emitted a warning when parsing the run config
		// 		continue
		// 	}

		// 	overrides[v.Config.StateKey] = v
		// }

		// Process each state key only once
		// refs, _ := v.GetReferences()
		// for _, ref := range refs {
		// 	subj, ok := ref.Subject.(addrs.Run)
		// 	if !ok {
		// 		continue
		// 	}

		// 	//look for the run with this address
		// 	for _, run := range t.opts.File.Runs {
		// 		if run.Config.Name == subj.Name {
		// 			g.Connect(dag.BasicEdge(cleanupMap[run.Name], node))
		// 			fmt.Printf("%s -> %s\n", node.Name(), cleanupMap[run.Name].Name())
		// 			break
		// 		}
		// 	}
		// }

		// Handle skip_cleanup attribute
		// switch {
		// // the node already has an applyOverride from a later run
		// case v.Config.SkipCleanup && override != nil && v.Config.SkipCleanupSet:
		// 	v.Diagnostics = v.Diagnostics.Append(tfdiags.Sourceless(
		// 		tfdiags.Warning,
		// 		"Multiple runs with skip_cleanup set",
		// 		fmt.Sprintf(`The run %q has skip_cleanup set to true, but shares state with a later run %q that also has skip_cleanup set. The later run takes precedence, and this attribute is ignored for the earlier run.`,
		// 			v.Config.Name, override.Config.Name),
		// 	))
		// case v.Config.SkipCleanup && override == nil:
		// 	node.applyOverride = v
		// }
	}

	// for node := range dag.SelectSeq(g.VerticesSeq(), func(*NodeStateCleanup) {}) {
	// 	if override, exists := overrides[node.run.Config.StateKey]; exists {
	// 		node.applyOverride = override
	// 	}
	// }

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

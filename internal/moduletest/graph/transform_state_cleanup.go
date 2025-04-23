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

// CleanupSubGraph is a subgraph that is responsible for cleaning up the state of
// resources defined in the state files created by the test runs.
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

	return Walk(g, ctx)
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

		// if skip_cleanup is set, we store the run in the overrideMap,
		// and the last run with this state key will be used to override the
		// state key in the cleanup node.
		if run.Config.SkipCleanup {
			overrideMap[run.Config.StateKey] = run
		}

		// Create a cleanup node for each run
		cleanupMap[run.Name] = &NodeStateCleanup{
			run:  run,
			opts: t.opts,
			deps: make(map[string]*NodeStateCleanup),
		}
		g.Add(cleanupMap[run.Name])
	}

	for _, run := range t.opts.File.Runs {
		node := cleanupMap[run.Name]
		// Ensure that referencer runs are cleaned up first
		refs, _ := run.GetReferences()
		for _, ref := range refs {
			subj, ok := ref.Subject.(addrs.Run)
			if !ok {
				continue
			}

			node.deps[subj.Name] = cleanupMap[subj.Name]

			// Look for the run with this address
			for _, r := range t.opts.File.Runs {
				if r.Config.Name == subj.Name {
					prev := cleanupMap[r.Name]
					g.Connect(dag.BasicEdge(prev, node))
					break
				}
			}
		}

		node.applyOverride = overrideMap[run.Config.StateKey]
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
				fmt.Printf("%s -> %s\n", prev.(*NodeStateCleanup).Name(), node.Name())
			}
			prev = node
			added[key] = true
		}
	}
	return nil
}

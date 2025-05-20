// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestRunTransformer is a GraphTransformer that adds all the test runs,
// and the variables defined in each run block, to the graph.
type TestRunTransformer struct {
	opts *graphOptions
}

func (t *TestRunTransformer) Transform(g *terraform.Graph) error {
	// Create and add nodes for each run
	var nodes []*NodeTestRun
	for _, run := range t.opts.File.Runs {
		node := &NodeTestRun{run: run, opts: t.opts}
		g.Add(node)
		nodes = append(nodes, node)
	}

	// Connect nodes based on dependencies
	if diags := t.connectDependencies(g, nodes); diags.HasErrors() {
		return tfdiags.DiagnosticsAsError{Diagnostics: diags}
	}

	// Runs with the same state key inherently depend on each other, so we
	// connect them sequentially.
	t.connectSameStateRuns(g, nodes)

	return nil
}

func (t *TestRunTransformer) connectDependencies(g *terraform.Graph, nodes []*NodeTestRun) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	nodeMap := make(map[string]*NodeTestRun)
	// add all nodes to the map. They are initialized to nil,
	// and we will update them as we iterate through the nodes in the next loop.
	for _, node := range nodes {
		nodeMap[node.run.Name] = nil
	}

	for _, node := range nodes {
		nodeMap[node.run.Name] = node // node encountered, so update the map

		// check for variable references
		varRefs := t.getVariableNames(node.run)

		refs, refDiags := node.run.GetReferences()
		if refDiags.HasErrors() {
			return diags.Append(refDiags)
		}
		for _, ref := range refs {
			switch subj := ref.Subject.(type) {
			case addrs.Run:
				dependency, ok := nodeMap[subj.Name]
				diagPrefix := "You can only reference run blocks that are in the same test file and will execute before the current run block."
				// Then this is a made up run block, and it doesn't exist at all.
				if !ok {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Reference to unknown run block",
						Detail:   fmt.Sprintf("The run block %q does not exist within this test file. %s", subj.Name, diagPrefix),
						Subject:  ref.SourceRange.ToHCL().Ptr(),
					})
					continue
				}

				// This run block exists, but it is after the current run block.
				if dependency == nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Reference to unavailable run block",
						Detail:   fmt.Sprintf("The run block %q has not executed yet. %s", subj.Name, diagPrefix),
						Subject:  ref.SourceRange.ToHCL().Ptr(),
					})
					continue
				}

				g.Connect(dag.BasicEdge(node, dependency))
			case addrs.InputVariable:
				if _, ok := varRefs[subj.Name]; !ok {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Reference to unavailable variable",
						Detail:   fmt.Sprintf("The input variable %q is not available to the current run block. You can only reference variables defined at the file or global levels.", subj.Name),
						Subject:  ref.SourceRange.ToHCL().Ptr(),
					})
				}
			}
		}
	}

	// If there is a run that has opted out of parallelism, we will connect it
	// sequentially to all previous and subsequent runs. This effectively
	// divides the parallelizable runs into separate groups, ensuring that
	// non-parallelizable runs are executed in sequence with respect to all
	// other runs.
	for i, node := range nodes {
		if node.run.Config.Parallel {
			continue
		}

		// Connect to all previous runs
		for j := 0; j < i; j++ {
			g.Connect(dag.BasicEdge(node, nodes[j]))
		}

		// Connect to all subsequent runs
		for j := i + 1; j < len(nodes); j++ {
			g.Connect(dag.BasicEdge(nodes[j], node))
		}
	}
	return diags
}

func (t *TestRunTransformer) connectSameStateRuns(g *terraform.Graph, nodes []*NodeTestRun) {
	stateRuns := make(map[string][]*NodeTestRun)
	for _, node := range nodes {
		key := node.run.GetStateKey()
		stateRuns[key] = append(stateRuns[key], node)
	}
	for _, runs := range stateRuns {
		for i := 1; i < len(runs); i++ {
			g.Connect(dag.BasicEdge(runs[i], runs[i-1]))
		}
	}
}

func (t *TestRunTransformer) getVariableNames(run *moduletest.Run) map[string]struct{} {
	set := make(map[string]struct{})
	for name := range t.opts.GlobalVars {
		set[name] = struct{}{}
	}
	for name := range run.Config.Variables {
		set[name] = struct{}{}
	}

	for name := range t.opts.File.Config.Variables {
		set[name] = struct{}{}
	}
	for name := range run.ModuleConfig.Module.Variables {
		set[name] = struct{}{}
	}
	return set
}

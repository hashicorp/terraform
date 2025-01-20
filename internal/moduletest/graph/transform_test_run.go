// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestRunTransformer is a GraphTransformer that adds all the test runs,
// and the variables defined in each run block, to the graph.
type TestRunTransformer struct {
	File       *moduletest.File
	globalVars map[string]backendrun.UnparsedVariableValue
}

func (t *TestRunTransformer) Transform(g *terraform.Graph) error {
	// Create and add nodes for each run
	nodes := t.createNodes(g)

	// Connect nodes based on dependencies
	if diags := t.connectDependencies(g, nodes); diags.HasErrors() {
		return tfdiags.NonFatalError{Diagnostics: diags}
	}

	// Connect nodes with the same state key sequentially
	t.connectStateKeyRuns(g, nodes)

	return nil
}

func (t *TestRunTransformer) createNodes(g *terraform.Graph) []*NodeTestRun {
	var nodes []*NodeTestRun
	var prev *NodeTestRun
	for _, run := range t.File.Runs {
		node := &NodeTestRun{run: run, file: t.File}
		g.Add(node)
		nodes = append(nodes, node)

		if prev != nil {
			parallelized := prev.run.Config.Parallel && run.Config.Parallel
			// we connect 2 sequential runs IF
			// 1. at least one of them is NOT eligible for parallelization OR
			// 2. they are both eligible for parallelization AND have the same state key
			if !parallelized || (parallelized && prev.run.GetStateKey() == run.GetStateKey()) {
				g.Connect(dag.BasicEdge(node, prev))
			}
		}
		prev = node
	}
	return nodes
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
	return diags
}

func (t *TestRunTransformer) connectStateKeyRuns(g *terraform.Graph, nodes []*NodeTestRun) {
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
	for name := range t.globalVars {
		set[name] = struct{}{}
	}
	for name := range run.Config.Variables {
		set[name] = struct{}{}
	}

	for name := range t.File.Config.Variables {
		set[name] = struct{}{}
	}
	for name := range run.ModuleConfig.Module.Variables {
		set[name] = struct{}{}
	}
	return set
}

// -------------------------------------------------------- CloseTestGraphTransformer --------------------------------------------------------

// CloseTestGraphTransformer is a GraphTransformer that adds a root to the graph.
type CloseTestGraphTransformer struct{}

func (t *CloseTestGraphTransformer) Transform(g *terraform.Graph) error {
	// close the root module
	closeRoot := &nodeCloseTest{}
	g.Add(closeRoot)

	// since this is closing the root module, make it depend on everything in
	// the root module.
	for _, v := range g.Vertices() {
		if v == closeRoot {
			continue
		}

		// since this is closing the root module,  and must be last, we can
		// connect to anything that doesn't have any up edges.
		if g.UpEdges(v).Len() == 0 {
			g.Connect(dag.BasicEdge(closeRoot, v))
		}
	}

	return nil
}

// This node doesn't do anything, it's just to ensure that we have a single
// root node that depends on everything in the root module.
type nodeCloseTest struct {
}

func (n *nodeCloseTest) Name() string {
	return "testroot"
}

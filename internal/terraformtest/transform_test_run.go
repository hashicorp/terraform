// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
)

// TestRunTransformer is a GraphTransformer that adds all the test runs to the graph.
type TestRunTransformer struct {
	File       *moduletest.File
	config     *configs.Config
	globalVars map[string]backendrun.UnparsedVariableValue
}

func (t *TestRunTransformer) Transform(g *terraform.Graph) error {
	prevs := make(map[string]*NodeTestRun)
	for _, run := range t.File.Runs {
		// If we're testing a specific configuration, we need to use that
		config := t.config
		if run.Config.ConfigUnderTest != nil {
			config = run.Config.ConfigUnderTest
		}

		node := &NodeTestRun{run: run, file: t.File, config: config, Module: config.Path}
		g.Add(node)

		// Connect the run to all the other runs that it depends on
		refs, _ := run.GetReferences()
		for _, ref := range refs {
			refRun, ok := ref.Subject.(addrs.Run)
			if !ok {
				continue
			}
			dependency, ok := prevs[refRun.Name]
			if !ok {
				// TODO: should we catch this error, or leave it, as it will still be caught in the test?
				return fmt.Errorf("dependency %q not found for run %q", refRun.Name, run.Name)
			}
			g.Connect(dag.BasicEdge(node, dependency))
		}
		prevs[run.Name] = node

		// Add all the variables that are defined in the run block
		for name, expr := range run.Config.Variables {
			variableNode := &nodeRunVariable{
				run:    run,
				Addr:   addrs.InputVariable{Name: name},
				Expr:   expr,
				config: config,
				Module: config.Path,
			}
			g.Add(variableNode)
			g.Connect(dag.BasicEdge(node, variableNode))
		}
	}

	return nil
}

// -------------------------------------------------------- CloseTestRootModuleTransformer --------------------------------------------------------

// CloseTestRootModuleTransformer is a GraphTransformer that adds a root to the graph.
type CloseTestRootModuleTransformer struct{}

func (t *CloseTestRootModuleTransformer) Transform(g *terraform.Graph) error {
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

// -------------------------------------------------------- ApplyNoParallelTransformer --------------------------------------------------------

// ApplyNoParallelTransformer ensures that all apply operations are run in sequential order.
// If we do not apply this transformer, the apply operations will be run in parallel, which
// may result in multiple runs acting on a particular resource at the same time.
type ApplyNoParallelTransformer struct{}

func (t *ApplyNoParallelTransformer) Transform(g *terraform.Graph) error {
	// find all the apply nodes
	runs := make([]*NodeTestRun, 0)
	for _, v := range g.Vertices() {
		run, ok := v.(*NodeTestRun)
		if !ok {
			continue
		}

		if run.run.Config.Command == configs.ApplyTestCommand {
			runs = append(runs, run)
		}
	}

	// sort them in descending order
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].run.Index > runs[j].run.Index
	})

	// connect them all in serial
	for i := 0; i < len(runs)-1; i++ {
		g.Connect(dag.BasicEdge(runs[i], runs[i+1]))
	}

	return nil
}

// ------------------------------------------ AttachVariablesTransformer -----------------------------------

// AttachVariablesTransformer is a GraphTransformer that ensures that each run node
// is connected to all the variable nodes in the graph.
// This is needed because this graph does not include the original terraform configuration
// which may reference variables declared in the test file.
type AttachVariablesTransformer struct {
}

func (t *AttachVariablesTransformer) Transform(g *terraform.Graph) error {
	t.attachVariableToTestRun(g)
	t.configVariablesToOthers(g)
	return nil
}

func (t *AttachVariablesTransformer) attachVariableToTestRun(g *terraform.Graph) {
	for _, v := range g.Vertices() {
		runNode, ok := v.(*NodeTestRun)
		if !ok {
			continue
		}

		for _, other := range g.Vertices() {
			if other == v {
				continue
			}

			switch other.(type) {
			case *nodeConfigVariable:
				if runNode.run == other.(*nodeConfigVariable).run {
					g.Connect(dag.BasicEdge(runNode, other))
				}
			case *nodeGlobalVariable:
				g.Connect(dag.BasicEdge(runNode, other))
				// if runNode.run == other.(*nodeGlobalVariable).run {
				// }
			case *nodeFileVariable:
				g.Connect(dag.BasicEdge(runNode, other))
			}
		}
	}
}

func (t *AttachVariablesTransformer) configVariablesToOthers(g *terraform.Graph) {
	for _, v := range g.Vertices() {
		node, ok := v.(*nodeConfigVariable)
		if !ok {
			continue
		}

		for _, other := range g.Vertices() {
			if _, ok := other.(*nodeConfigVariable); ok {
				continue
			}

			switch other := other.(type) {
			case *nodeFileVariable:
				if node.config.Module.SourceDir == other.config.Module.SourceDir {
					g.Connect(dag.BasicEdge(node, other))
				}
			case *nodeGlobalVariable:
				show := node.config.Module.SourceDir == other.config.Module.SourceDir
				_, ok := node.config.Module.Variables[other.Addr.Name]
				show = show || ok
				if show {
					g.Connect(dag.BasicEdge(node, other))
				}
			case *nodeRunVariable:
				if node.config.Module.SourceDir == other.config.Module.SourceDir {
					g.Connect(dag.BasicEdge(node, other))
				}
			}
		}
	}
}

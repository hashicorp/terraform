// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"fmt"

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
	runsSoFar := make(map[string]*NodeTestRun)
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
			dependency, ok := runsSoFar[refRun.Name]
			if !ok {
				// TODO: should we catch this error, or leave it, as it will still be caught in the test?
				return fmt.Errorf("dependency %q not found for run %q", refRun.Name, run.Name)
			}
			g.Connect(dag.BasicEdge(node, dependency))
		}
		runsSoFar[run.Name] = node

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

// ------------------------------------------ AttachVariablesToTestRunTransformer -----------------------------------

// AttachVariablesToTestRunTransformer is a GraphTransformer that ensures that each run node
// is connected to all the variable nodes in the graph.
// This is needed because this graph does not include the original terraform configuration
// which may reference variables declared in the test file.
type AttachVariablesToTestRunTransformer struct {
}

func refsMap(n terraform.GraphNodeReferencer) map[string]*addrs.Reference {
	result := make(map[string]*addrs.Reference)
	for _, ref := range n.References() {
		result[ref.Subject.String()] = ref
	}
	return result
}

func (t *AttachVariablesToTestRunTransformer) Transform(g *terraform.Graph) error {
	for _, v := range g.Vertices() {
		runNode, ok := v.(*NodeTestRun)
		if !ok {
			continue
		}
		runNodeRefs := runNode.refsMap()

		for _, other := range g.Vertices() {
			if other == v {
				continue
			}

			switch other := other.(type) {
			case *nodeConfigVariable:
				if runNode.run == other.run {
					g.Connect(dag.BasicEdge(runNode, other))
				}
			case *nodeGlobalVariable:
				key := fmt.Sprintf("var.%s", other.Addr.Name)
				// connect only if the global variable is referenced in the run block
				if _, ok := runNodeRefs[key]; ok {
					g.Connect(dag.BasicEdge(runNode, other))
				}
			case *nodeFileVariable:
				g.Connect(dag.BasicEdge(runNode, other))
			}
		}
	}

	// referencing global variables from the all nodes that use it
	for _, v := range g.Vertices() {
		if _, ok := v.(*NodeTestRun); ok {
			continue
		}
		node, ok := v.(terraform.GraphNodeReferencer)
		if !ok {
			continue
		}
		nodeRefs := refsMap(node)

		for _, other := range g.Vertices() {
			if other == v {
				continue
			}

			switch other := other.(type) {
			case *nodeGlobalVariable:
				// key if the global variable is referenced in the config
				cfgKey := other.Addr.Name
				if _, ok := nodeRefs[cfgKey]; ok {
					g.Connect(dag.BasicEdge(node, other))
					continue
				}
				// key if the global variable is referenced in the run block
				key := fmt.Sprintf("var.%s", other.Addr.Name)
				if _, ok := nodeRefs[key]; ok {
					g.Connect(dag.BasicEdge(node, other))
				}
			}
		}
	}

	// referencing global variables from the config variables that use it
	for _, v := range g.Vertices() {
		if _, ok := v.(*NodeTestRun); ok {
			continue
		}
		cfgNode, ok := v.(*nodeConfigVariable)
		if !ok {
			continue
		}

		for _, other := range g.Vertices() {
			if other == v {
				continue
			}

			switch other := other.(type) {
			case *nodeGlobalVariable:
				if cfgNode.variable.Name == other.Addr.Name {
					g.Connect(dag.BasicEdge(cfgNode, other))
				}
			}
		}
	}
	return nil
}

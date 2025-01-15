// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
)

// VariablesTransformer is a GraphTransformer that adds the config variables and global variables to the graph.
type VariablesTransformer struct {
	File       *moduletest.File
	config     *configs.Config
	globalVars map[string]backendrun.UnparsedVariableValue
}

func (t *VariablesTransformer) Transform(g *terraform.Graph) error {
	modeMap := make(map[string]configs.VariableParsingMode)
	// For each run, we have to create a node for each variable
	// in the module configuration that the run uses.
	for _, run := range t.File.Runs {
		config := t.config

		// if the run has reference to a specific module configuration, we use that
		if run.Config.ConfigUnderTest != nil {
			config = run.Config.ConfigUnderTest
		}
		for _, v := range config.Module.Variables {
			node := &nodeConfigVariable{
				Addr:     addrs.InputVariable{Name: v.Name},
				variable: v,
				run:      run,
				Module:   config.Path,
				config:   config,
			}
			g.Add(node)
		}

		// For all configurations used in the test (the main module configuration
		// and any other module configurations referenced in runs), check if the
		// global variables are used in any of them. If they are, store the parsing
		// mode of the variable in the modeMap.
		// TODO: What happens if 2 configurations use the same global variable but with different parsing modes?
		for name := range t.globalVars {
			if variable, ok := config.Module.Variables[name]; ok {
				modeMap[name] = variable.ParsingMode
			}
		}

	}

	// Add the global variables to the graph
	for name, unparsed := range t.globalVars {
		parsingMode := configs.VariableParseHCL
		if _, exists := modeMap[name]; exists {
			parsingMode = modeMap[name]
		}

		node := &nodeGlobalVariable{
			Addr:        addrs.InputVariable{Name: name},
			unparsed:    unparsed,
			parsingMode: parsingMode,
			config:      t.config,
			Module:      t.config.Path,
		}
		g.Add(node)
	}
	return nil
}

// RemoveDanglingGlobalTransformer is a GraphTransformer that removes dangling global variables from the graph.
type RemoveDanglingGlobalTransformer struct {
}

func (t *RemoveDanglingGlobalTransformer) Transform(g *terraform.Graph) error {
	for _, v := range g.Vertices() {
		node, ok := v.(*nodeGlobalVariable)
		if !ok {
			continue
		}

		set := g.UpEdges(node)
		if len(set) == 0 {
			g.Remove(node)
		}
	}
	return nil
}

// RemoveInvalidRunVarsTransformer is a GraphTransformer that removes invalid run variables connections from the graph.
// runs should only have connections to variables that are in the same run.
type RemoveInvalidRunVarsTransformer struct {
}

func (t *RemoveInvalidRunVarsTransformer) Transform(g *terraform.Graph) error {
	for _, v := range g.Vertices() {
		node, ok := v.(*NodeTestRun)
		if !ok {
			continue
		}

		set := g.DownEdges(node)
		for _, dep := range set {
			if runVar, ok := dep.(*nodeRunVariable); ok && runVar.run != node.run {
				g.RemoveEdge(dag.BasicEdge(node, dep))
			}
		}
	}
	return nil
}

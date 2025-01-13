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

// ConfigTransformer is a GraphTransformer that adds all the test runs to the graph.
type ConfigTransformer struct {
	File       *moduletest.File
	config     *configs.Config
	globalVars map[string]backendrun.UnparsedVariableValue
}

func (t *ConfigTransformer) Transform(g *terraform.Graph) error {
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

type TransformGlobal struct {
	File       *moduletest.File
	config     *configs.Config
	globalVars map[string]backendrun.UnparsedVariableValue
}

func (t *TransformGlobal) Transform(g *terraform.Graph) error {
	// modeMap := make(map[string]configs.VariableParsingMode)
	// for _, run := range t.File.Runs {
	// 	config := t.config

	// 	// if the run has reference to a specific module configuration, we use that
	// 	if run.Config.ConfigUnderTest != nil {
	// 		config = run.Config.ConfigUnderTest
	// 	}
	// 	// For all configurations used in the test (the main module configuration
	// 	// and any other module configurations referenced in runs), check if the
	// 	// global variables are used in any of them. If they are, store the parsing
	// 	// mode of the variable in the modeMap.
	// 	// TODO: What happens if 2 configurations use the same global variable but with different parsing modes?
	// 	for name := range t.globalVars {
	// 		if variable, ok := config.Module.Variables[name]; ok {
	// 			modeMap[name] = variable.ParsingMode
	// 		}
	// 	}

	// }

	// // Add the global variables to the graph
	// for name, unparsed := range t.globalVars {
	// 	parsingMode := configs.VariableParseHCL
	// 	if _, exists := modeMap[name]; exists {
	// 		parsingMode = modeMap[name]
	// 	}

	// 	node := &nodeGlobalVariable{
	// 		Addr:        addrs.InputVariable{Name: name},
	// 		unparsed:    unparsed,
	// 		parsingMode: parsingMode,
	// 		config:      t.config,
	// 		Module:      t.config.Path,
	// 	}
	// 	g.Add(node)
	// }

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

// RemoveInvalidRuns is a GraphTransformer that removes invalid runs from the graph.
// For each run, it removes all the run variables that are not related to the run.
type RemoveInvalidRuns struct {
}

func (t *RemoveInvalidRuns) Transform(g *terraform.Graph) error {
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

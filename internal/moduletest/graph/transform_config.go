// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestConfigTransformer is a GraphTransformer that adds all the test runs,
// and the variables defined in each run block, to the graph.
type TestConfigTransformer struct{}

func (t *TestConfigTransformer) Transform(g *terraform.Graph) error {
	// This map tracks the state of each run in the file. If multiple runs
	// have the same state key, they will share the same state.
	statesMap := make(map[string]*TestFileState)
	for _, v := range g.Vertices() {
		node, ok := v.(*NodeTestRun)
		if !ok {
			continue
		}
		if _, exists := statesMap[node.run.GetStateKey()]; !exists {
			statesMap[node.run.GetStateKey()] = &TestFileState{
				Run:   nil,
				State: states.NewState(),
			}
		}
	}
	cfgNode := &nodeConfig{configMap: statesMap}
	g.Add(cfgNode)

	// Connect all the test runs to the config node, so that the config node
	// is executed before any of the test runs.
	for _, v := range g.Vertices() {
		node, ok := v.(*NodeTestRun)
		if !ok {
			continue
		}
		g.Connect(dag.BasicEdge(node, cfgNode))
	}

	return nil
}

type nodeConfig struct {
	configMap map[string]*TestFileState
}

func (n *nodeConfig) Name() string {
	return "nodeConfig"
}

type GraphNodeExecutable interface {
	Execute(ctx *EvalContext) tfdiags.Diagnostics
}

func (n *nodeConfig) Execute(ctx *EvalContext) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	ctx.FileStates = n.configMap
	return diags
}

// TestFileState is a helper struct that just maps a run block to the state that
// was produced by the execution of that run block.
type TestFileState struct {
	Run   *moduletest.Run
	State *states.State
}

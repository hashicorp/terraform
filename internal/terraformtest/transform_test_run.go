// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
)

// TestRunTransformer is a GraphTransformer that adds all the test runs,
// and the variables defined in each run block, to the graph.
type TestRunTransformer struct {
	File       *moduletest.File
	config     *configs.Config
	globalVars map[string]backendrun.UnparsedVariableValue
}

func (t *TestRunTransformer) Transform(g *terraform.Graph) error {
	var prev *NodeTestRun
	for _, run := range t.File.Runs {
		node := &NodeTestRun{run: run, file: t.File}
		g.Add(node)
		if prev != nil {
			g.Connect(dag.BasicEdge(node, prev))
		}
		prev = node
	}

	return nil
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

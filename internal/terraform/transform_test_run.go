// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
)

type TestRunTransformer struct {
	File *moduletest.File
}

func (t *TestRunTransformer) Transform(g *Graph) error {
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

// -------------------------------------------------------- CloseTestRootModuleTransformer --------------------------------------------------------

// CloseTestGraphTransformer is a GraphTransformer that adds a root to the graph.
type CloseTestGraphTransformer struct{}

func (t *CloseTestGraphTransformer) Transform(g *Graph) error {
	// close the test graph
	closeRoot := &nodeCloseTest{}
	g.Add(closeRoot)

	// since this is closing the node, make it depend on every run
	for _, v := range g.Vertices() {
		if v == closeRoot {
			continue
		}

		// since this is closing the node,  and must be last, we can
		// connect to anything that doesn't have any up edges.
		if g.UpEdges(v).Len() == 0 {
			g.Connect(dag.BasicEdge(closeRoot, v))
		}
	}

	return nil
}

// This node doesn't do anything, it's just to ensure that we have a single
// root node that depends on everything in the test file.
type nodeCloseTest struct{}

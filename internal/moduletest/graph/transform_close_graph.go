// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/terraform"
)

// CloseTestGraphTransformer is a GraphTransformer that adds a closing node to the graph.
type CloseTestGraphTransformer struct{}

func (t *CloseTestGraphTransformer) Transform(g *terraform.Graph) error {
	closeRoot := &nodeCloseTest{}
	g.Add(closeRoot)

	for _, v := range g.Vertices() {
		if v == closeRoot {
			continue
		}

		// since this is closing the graph, make it depend on everything in
		// the graph that does not have a parent. Such nodes are the real roots
		// of the graph, and since they are now siblings of the closing root node,
		// they are allowed to run in parallel.
		if g.UpEdges(v).Len() == 0 {
			g.Connect(dag.BasicEdge(closeRoot, v))
		}
	}

	return nil
}

// This node doesn't do anything, it's just to ensure that we have a single
// root node that depends on everything in the graph. The nodes that it depends
// on are the real roots of the graph.
type nodeCloseTest struct {
}

func (n *nodeCloseTest) Name() string {
	return "testcloser"
}

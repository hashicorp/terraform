// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"github.com/hashicorp/terraform/internal/dag"
)

const rootNodeName = "root"

// RootTransformer is a GraphTransformer that adds a root to the graph.
type RootTransformer struct{}

func (t *RootTransformer) Transform(g *Graph) error {
	addRootNodeToGraph(g)
	return nil
}

// addRootNodeToGraph modifies the given graph in-place so that it has a root
// node if it didn't already have one and so that any other node which doesn't
// already depend on something will depend on that root node.
//
// After this function returns, the graph will have only one node that doesn't
// depend on any other nodes.
func addRootNodeToGraph(g *Graph) {
	// We always add the root node. This is a singleton so if it's already
	// in the graph this will do nothing and just retain the existing root node.
	//
	// Note that rootNode is intentionally added by value and not by pointer
	// so that all root nodes will be equal to one another and therefore
	// coalesce when two valid graphs get merged together into a single graph.
	g.Add(rootNode)

	// Everything that doesn't already depend on at least one other node will
	// depend on the root node, except the root node itself.
	for _, v := range g.Vertices() {
		if v == dag.Vertex(rootNode) {
			continue
		}

		if g.UpEdges(v).Len() == 0 {
			g.Connect(dag.BasicEdge(rootNode, v))
		}
	}
}

type graphNodeRoot struct{}

// rootNode is the singleton value representing all root graph nodes.
//
// The root node for all graphs should be this value directly, and in particular
// _not_ a pointer to this value. Using the value directly here means that
// multiple root nodes will always coalesce together when subsuming one graph
// into another.
var rootNode graphNodeRoot

func (n graphNodeRoot) Name() string {
	return rootNodeName
}

// CloseRootModuleTransformer is a GraphTransformer that adds a root to the graph.
type CloseRootModuleTransformer struct{}

func (t *CloseRootModuleTransformer) Transform(g *Graph) error {
	// close the root module
	closeRoot := &nodeCloseModule{}
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

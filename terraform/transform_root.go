package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

const rootNodeName = "root"

// RootTransformer is a GraphTransformer that adds a root to the graph.
type RootTransformer struct{}

func (t *RootTransformer) Transform(g *Graph) error {
	// If we already have a good root, we're done
	if _, err := g.Root(); err == nil {
		return nil
	}

	// Add a root
	var root graphNodeRoot
	g.Add(root)

	// Connect the root to all the edges that need it
	for _, v := range g.Vertices() {
		if v == root {
			continue
		}

		if g.UpEdges(v).Len() == 0 {
			g.Connect(dag.BasicEdge(root, v))
		}
	}

	return nil
}

type graphNodeRoot struct{}

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

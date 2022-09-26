package terraform

import (
	"github.com/hashicorp/terraform/internal/dag"
)

const rootNodeName = "root"

// RootTransformer is a GraphTransformer that adds a root to the graph.
type RootTransformer struct{}

func (t *RootTransformer) Transform(g *Graph) error {
	// If we already have a good root, we're done
	if _, err := g.Root(); err == nil {
		return nil
	}

	// We intentionally add a graphNodeRoot value -- rather than a pointer to
	// one -- so that all root nodes will coalesce together if two graphs
	// are merged. Each distinct node value can only be in a graph once,
	// so adding another graphNodeRoot value to the same graph later will
	// be a no-op and all of the edges from root nodes will coalesce together
	// under Graph.Subsume.
	//
	// It's important to retain this coalescing guarantee under future
	// maintenence.
	var root graphNodeRoot
	g.Add(root)

	// We initially make the root node depend on every node except itself.
	// If the caller subsequently runs transitive reduction on the graph then
	// it's typical for some of these edges to then be removed.
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

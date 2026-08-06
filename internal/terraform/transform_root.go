// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

// CloseRootModuleTransformer is a GraphTransformer that adds a root to the graph.
type CloseRootModuleTransformer struct{}

func (t *CloseRootModuleTransformer) Transform(g *Graph) error {
	// close the root module
	closeRoot := &nodeCloseModule{}
	g.Add(closeRoot)

	// since this is closing the root module, make it depend on everything in
	// the root module.
	for v := range g.VerticesSeq() {
		if v == closeRoot {
			continue
		}

		// since this is closing the root module,  and must be last, we can
		// connect to anything that doesn't have any up edges.
		if g.EdgesTo(v).Len() == 0 {
			g.Connect(closeRoot, v)
		}
	}

	return nil
}

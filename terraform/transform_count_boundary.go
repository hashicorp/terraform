package terraform

import (
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
)

// CountBoundaryTransformer adds a node that depends on everything else
// so that it runs last in order to clean up the state for nodes that
// are on the "count boundary": "foo.0" when only one exists becomes "foo"
type CountBoundaryTransformer struct {
	Config *configs.Config
}

func (t *CountBoundaryTransformer) Transform(g *Graph) error {
	node := &NodeCountBoundary{
		Config: t.Config,
	}
	g.Add(node)

	// Depends on everything
	for _, v := range g.Vertices() {
		// Don't connect to ourselves
		if v == node {
			continue
		}

		// Connect!
		g.Connect(dag.BasicEdge(node, v))
	}

	return nil
}

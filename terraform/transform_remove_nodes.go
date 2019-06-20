package terraform

import (
	"log"

	"github.com/hashicorp/terraform/dag"
)

// RemoveNodesTransformer is a GraphTransformer that removes any nodes which
// fail a caller-provided test, but preserves the connectivity of all of the
// remaining nodes so that their relative ordering stays the same.
type RemoveNodesTransformer struct {
	ShouldKeep func(v dag.Vertex) bool
}

var _ GraphTransformer = (*RemoveNodesTransformer)(nil)

// Transform implements GraphTransformer.
func (t *RemoveNodesTransformer) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		if !t.ShouldKeep(v) {
			log.Printf("[TRACE] RemoveNodesTransformer: removing %s", dag.VertexName(v))
			g.RemovePreservingConnectivity(v)
		}
	}
	return nil
}

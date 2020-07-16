package terraform

import (
	"log"

	"github.com/hashicorp/terraform/dag"
)

// edgesBetweenModuleInstancesTransformer is a graph transformer that
// compensates for the fact that our earlier graph transforms in the
// apply graph builder are often very conservative in that they do their
// analyses based on relationships between whole resources (unexpanded)
// rather than individual module instances.
//
// When module expansion is in play, this can cause there to be dependency
// edges between resource instances in different instances of the same module,
// but those edges can never be necessary in practice because different
// instances of a module can interact with each other only indirectly.
//
// This transformer is specifically for the apply graph because it relies on
// the fact that in the apply graph each individual resource instance is
// represented by a separate graph node. In other graph types, we only have
// one node per yet-to-be-expanded resource.
//
// In the longer term we'd like to redesign the apply graph build process to
// be simpler and more precise, but we're using this post-processing fixup for
// now as a stop-gap to defer a more risky rewrite or refactor.
type edgesBetweenModuleInstancesTransformer struct {
}

func (t edgesBetweenModuleInstancesTransformer) Transform(g *Graph) error {
	// The simple filtering rule for this transformer is to remove any
	// edge where both nodes are already-expanded resource instance nodes
	// and where their fully-expanded addresses have different module instance
	// paths.

	for _, edge := range g.Edges() {
		source, sourceOk := edge.Source().(GraphNodeResourceInstance)
		target, targetOk := edge.Target().(GraphNodeResourceInstance)
		if !(sourceOk && targetOk) {
			// We only operate on resource instance nodes.
			continue
		}

		sourceAddr := source.ResourceInstanceAddr()
		targetAddr := target.ResourceInstanceAddr()
		if !targetAddr.Module.Equal(sourceAddr.Module) {
			// Direct edges between resources in different modules are always
			// unnecessary, so we remove them to make the graph more accurate
			// and thus improve concurrency and reduce the risk of unnecessary
			// cycles.
			log.Printf("[TRACE] edgesBetweenModuleInstancesTransformer: removing unnecessary dependency %s -> %s", dag.VertexName(source), dag.VertexName(target))
			g.RemoveEdge(edge)
		}
	}

	return nil
}

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// OutputTransformer is a GraphTransformer that adds all the outputs
// in the configuration to the graph.
//
// This is done for the apply graph builder even if dependent nodes
// aren't changing since there is no downside: the state will be available
// even if the dependent items aren't changing.
type OutputTransformer struct {
	Module *module.Tree
}

func (t *OutputTransformer) Transform(g *Graph) error {
	return t.transform(g, t.Module)
}

func (t *OutputTransformer) transform(g *Graph, m *module.Tree) error {
	// If no config, no outputs
	if m == nil {
		return nil
	}

	// Transform all the children. We must do this first because
	// we can reference module outputs and they must show up in the
	// reference map.
	for _, c := range m.Children() {
		if err := t.transform(g, c); err != nil {
			return err
		}
	}

	// If we have no outputs, we're done!
	os := m.Config().Outputs
	if len(os) == 0 {
		return nil
	}

	// Build the reference map so we can determine if we're referencing things.
	refMap := NewReferenceMap(g.Vertices())

	// Add all outputs here
	for _, o := range os {
		// Build the node
		node := &NodeApplyableOutput{
			PathValue: normalizeModulePath(m.Path()),
			Config:    o,
		}

		// If the node references something, then we check to make sure
		// that the thing it references is in the graph. If it isn't, then
		// we don't add it because we may not be able to compute the output.
		//
		// If the node references nothing, we always include it since there
		// is no other clear time to compute it.
		matches, missing := refMap.References(node)
		if len(missing) > 0 {
			log.Printf(
				"[INFO] Not including %q in graph, matches: %v, missing: %s",
				dag.VertexName(node), matches, missing)
			continue
		}

		// Add it!
		g.Add(node)
	}

	return nil
}

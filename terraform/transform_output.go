package terraform

import (
	"github.com/hashicorp/terraform/config/module"
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

	// Add all outputs here
	for _, o := range os {
		// Build the node.
		//
		// NOTE: For now this is just an "applyable" output. As we build
		// new graph builders for the other operations I suspect we'll
		// find a way to parameterize this, require new transforms, etc.
		node := &NodeApplyableOutput{
			PathValue: normalizeModulePath(m.Path()),
			Config:    o,
		}

		// Add it!
		g.Add(node)
	}

	return nil
}

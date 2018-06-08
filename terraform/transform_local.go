package terraform

import (
	"github.com/hashicorp/terraform/config/module"
)

// LocalTransformer is a GraphTransformer that adds all the local values
// from the configuration to the graph.
type LocalTransformer struct {
	Module *module.Tree
}

func (t *LocalTransformer) Transform(g *Graph) error {
	return t.transformModule(g, t.Module)
}

func (t *LocalTransformer) transformModule(g *Graph, m *module.Tree) error {
	if m == nil {
		// Can't have any locals if there's no config
		return nil
	}

	for _, local := range m.Config().Locals {
		node := &NodeLocal{
			PathValue: normalizeModulePath(m.Path()),
			Config:    local,
		}

		g.Add(node)
	}

	// Also populate locals for child modules
	for _, c := range m.Children() {
		if err := t.transformModule(g, c); err != nil {
			return err
		}
	}

	return nil
}

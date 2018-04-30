package terraform

import (
	"github.com/hashicorp/terraform/configs"
)

// LocalTransformer is a GraphTransformer that adds all the local values
// from the configuration to the graph.
type LocalTransformer struct {
	Config *configs.Config
}

func (t *LocalTransformer) Transform(g *Graph) error {
	return t.transformModule(g, t.Config)
}

func (t *LocalTransformer) transformModule(g *Graph, c *configs.Config) error {
	if c == nil {
		// Can't have any locals if there's no config
		return nil
	}

	// Our addressing system distinguishes between modules and module instances,
	// but we're not yet ready to make that distinction here (since we don't
	// support "count"/"for_each" on modules) and so we just do a naive
	// transform of the module path into a module instance path, assuming that
	// no keys are in use. This should be removed when "count" and "for_each"
	// are implemented for modules.
	path := c.Path.UnkeyedInstanceShim()

	for _, local := range c.Module.Locals {
		addr := path.LocalValue(local.Name)
		node := &NodeLocal{
			Addr:   addr,
			Config: local,
		}
		g.Add(node)
	}

	// Also populate locals for child modules
	for _, cc := range c.Children {
		if err := t.transformModule(g, cc); err != nil {
			return err
		}
	}

	return nil
}

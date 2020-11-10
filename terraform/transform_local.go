package terraform

import (
	"github.com/hashicorp/terraform/addrs"
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

	for _, local := range c.Module.Locals {
		addr := addrs.LocalValue{Name: local.Name}
		node := &nodeExpandLocal{
			Addr:   addr,
			Module: c.Path,
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

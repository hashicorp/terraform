package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/states"
)

// OutputTransformer is a GraphTransformer that adds nodes for all outputs
// that exist in the configuration or the state, allowing us to evaluate new
// values for them or remove them altogether.
//
// The caller must provide a factory functions for constructing both evaluate
// nodes and destroy nodes, allowing different concrete node types to be used
// during different graph walks. For destroy nodes, the given configuration
// is nil.
type OutputTransformer struct {
	Config *configs.Config
	State  *states.State

	NewNode func(addr addrs.AbsOutputValue, config *configs.Output) dag.Vertex
}

func (t *OutputTransformer) Transform(g *Graph) error {
	err := t.transformConfig(g, t.Config)
	if err != nil {
		return err
	}

	return t.transformOrphans(g)
}

func (t *OutputTransformer) transformConfig(g *Graph, c *configs.Config) error {
	// If we have no config then there can be no outputs.
	if c == nil {
		return nil
	}

	// Transform all the children. We must do this first because
	// we can reference module outputs and they must show up in the
	// reference map.
	for _, cc := range c.Children {
		if err := t.transformConfig(g, cc); err != nil {
			return err
		}
	}

	// Our addressing system distinguishes between modules and module instances,
	// but we're not yet ready to make that distinction here (since we don't
	// support "count"/"for_each" on modules) and so we just do a naive
	// transform of the module path into a module instance path, assuming that
	// no keys are in use. This should be removed when "count" and "for_each"
	// are implemented for modules.
	path := c.Path.UnkeyedInstanceShim()

	for _, o := range c.Module.Outputs {
		addr := path.OutputValue(o.Name)
		node := t.NewNode(addr, o)
		g.Add(node)
	}

	return nil
}

func (t *OutputTransformer) transformOrphans(g *Graph) error {
	// "orphans" here are any outputs present in the state that are not
	// present in the configuration, which we'll therefore need to remove from
	// the state after apply.

	for _, ms := range t.State.Modules {
		cfg := t.Config.DescendentForInstance(ms.Addr)
		for name := range ms.OutputValues {
			addr := addrs.OutputValue{Name: name}.Absolute(ms.Addr)
			n := t.NewNode(addr, nil)
			if cfg == nil {
				g.Add(n)
			} else if _, exists := cfg.Module.Outputs[name]; !exists {
				g.Add(n)
			}
		}
	}

	return nil
}

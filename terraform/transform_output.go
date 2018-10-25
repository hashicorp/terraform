package terraform

import (
	"log"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
)

// OutputTransformer is a GraphTransformer that adds all the outputs
// in the configuration to the graph.
//
// This is done for the apply graph builder even if dependent nodes
// aren't changing since there is no downside: the state will be available
// even if the dependent items aren't changing.
type OutputTransformer struct {
	Config *configs.Config
}

func (t *OutputTransformer) Transform(g *Graph) error {
	return t.transform(g, t.Config)
}

func (t *OutputTransformer) transform(g *Graph, c *configs.Config) error {
	// If we have no config then there can be no outputs.
	if c == nil {
		return nil
	}

	// Transform all the children. We must do this first because
	// we can reference module outputs and they must show up in the
	// reference map.
	for _, cc := range c.Children {
		if err := t.transform(g, cc); err != nil {
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
		node := &NodeApplyableOutput{
			Addr:   addr,
			Config: o,
		}
		g.Add(node)
	}

	return nil
}

// DestroyOutputTransformer is a GraphTransformer that adds nodes to delete
// outputs during destroy. We need to do this to ensure that no stale outputs
// are ever left in the state.
type DestroyOutputTransformer struct {
}

func (t *DestroyOutputTransformer) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		output, ok := v.(*NodeApplyableOutput)
		if !ok {
			continue
		}

		// create the destroy node for this output
		node := &NodeDestroyableOutput{
			Addr:   output.Addr,
			Config: output.Config,
		}

		log.Printf("[TRACE] creating %s", node.Name())
		g.Add(node)

		deps, err := g.Descendents(v)
		if err != nil {
			return err
		}

		// the destroy node must depend on the eval node
		deps.Add(v)

		for _, d := range deps.List() {
			log.Printf("[TRACE] %s depends on %s", node.Name(), dag.VertexName(d))
			g.Connect(dag.BasicEdge(node, d))
		}
	}
	return nil
}

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/addrs"
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

	// Add plannable outputs to the graph, which will be dynamically expanded
	// into NodeApplyableOutputs to reflect possible expansion
	// through the presence of "count" or "for_each" on the modules.
	for _, o := range c.Module.Outputs {
		node := &nodeExpandOutput{
			Addr:   addrs.OutputValue{Name: o.Name},
			Module: c.Path,
			Config: o,
		}
		log.Printf("[TRACE] OutputTransformer: adding %s as %T", o.Name, node)
		g.Add(node)
	}

	return nil
}

// destroyRootOutputTransformer is a GraphTransformer that adds nodes to delete
// outputs during destroy. We need to do this to ensure that no stale outputs
// are ever left in the state.
type destroyRootOutputTransformer struct {
	Destroy bool
}

func (t *destroyRootOutputTransformer) Transform(g *Graph) error {
	// Only clean root outputs on a full destroy
	if !t.Destroy {
		return nil
	}

	for _, v := range g.Vertices() {
		output, ok := v.(*nodeExpandOutput)
		if !ok {
			continue
		}

		// We only destroy root outputs
		if !output.Module.Equal(addrs.RootModule) {
			continue
		}

		// create the destroy node for this output
		node := &NodeDestroyableOutput{
			Addr:   output.Addr.Absolute(addrs.RootModuleInstance),
			Config: output.Config,
		}

		log.Printf("[TRACE] creating %s", node.Name())
		g.Add(node)

		deps := g.UpEdges(v)

		for _, d := range deps {
			log.Printf("[TRACE] %s depends on %s", node.Name(), dag.VertexName(d))
			g.Connect(dag.BasicEdge(node, d))
		}

		// We no longer need the expand node, since we intend to remove this
		// output from the state.
		g.Remove(v)
	}
	return nil
}

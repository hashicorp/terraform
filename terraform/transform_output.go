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

	// Add all outputs here
	for _, o := range os {
		node := &NodeApplyableOutput{
			PathValue: normalizeModulePath(m.Path()),
			Config:    o,
		}

		// Add it!
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
			PathValue: output.PathValue,
			Config:    output.Config,
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

package terraform

import (
	"errors"

	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// FlatConfigTransformer is a GraphTransformer that adds the configuration
// to the graph. The module used to configure this transformer must be
// the root module.
//
// This transform adds the nodes but doesn't connect any of the references.
// The ReferenceTransformer should be used for that.
//
// NOTE: In relation to ConfigTransformer: this is a newer generation config
// transformer. It puts the _entire_ config into the graph (there is no
// "flattening" step as before).
type FlatConfigTransformer struct {
	Concrete ConcreteResourceNodeFunc // What to turn resources into

	Module *module.Tree
}

func (t *FlatConfigTransformer) Transform(g *Graph) error {
	// If no module, we do nothing
	if t.Module == nil {
		return nil
	}

	// If the module is not loaded, that is an error
	if !t.Module.Loaded() {
		return errors.New("module must be loaded")
	}

	return t.transform(g, t.Module)
}

func (t *FlatConfigTransformer) transform(g *Graph, m *module.Tree) error {
	// If no module, no problem
	if m == nil {
		return nil
	}

	// Transform all the children.
	for _, c := range m.Children() {
		if err := t.transform(g, c); err != nil {
			return err
		}
	}

	// Get the configuration for this module
	config := m.Config()

	// Write all the resources out
	for _, r := range config.Resources {
		// Grab the address for this resource
		addr, err := parseResourceAddressConfig(r)
		if err != nil {
			return err
		}
		addr.Path = m.Path()

		// Build the abstract resource. We have the config already so
		// we'll just pre-populate that.
		abstract := &NodeAbstractResource{
			Addr:   addr,
			Config: r,
		}
		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}

		g.Add(node)
	}

	return nil
}

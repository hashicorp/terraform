package terraform

import (
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// ConfigTransformer is a GraphTransformer that adds all the resources
// from the configuration to the graph.
//
// The module used to configure this transformer must be the root module.
//
// Only resources are added to the graph. Variables, outputs, and
// providers must be added via other transforms.
//
// Unlike ConfigTransformerOld, this transformer creates a graph with
// all resources including module resources, rather than creating module
// nodes that are then "flattened".
type ConfigTransformer struct {
	Concrete ConcreteResourceNodeFunc

	Module *module.Tree
}

func (t *ConfigTransformer) Transform(g *Graph) error {
	// If no module is given, we don't do anything
	if t.Module == nil {
		return nil
	}

	// If the module isn't loaded, that is simply an error
	if !t.Module.Loaded() {
		return errors.New("module must be loaded for ConfigTransformer")
	}

	// Start the transformation process
	return t.transform(g, t.Module)
}

func (t *ConfigTransformer) transform(g *Graph, m *module.Tree) error {
	// If no config, do nothing
	if m == nil {
		return nil
	}

	// Add our resources
	if err := t.transformSingle(g, m); err != nil {
		return err
	}

	// Transform all the children.
	for _, c := range m.Children() {
		if err := t.transform(g, c); err != nil {
			return err
		}
	}

	return nil
}

func (t *ConfigTransformer) transformSingle(g *Graph, m *module.Tree) error {
	log.Printf("[TRACE] ConfigTransformer: Starting for path: %v", m.Path())

	// Get the configuration for this module
	config := m.Config()

	// Build the path we're at
	path := m.Path()

	// Write all the resources out
	for _, r := range config.Resources {
		// Build the resource address
		addr, err := parseResourceAddressConfig(r)
		if err != nil {
			panic(fmt.Sprintf(
				"Error parsing config address, this is a bug: %#v", r))
		}
		addr.Path = path

		// Build the abstract node and the concrete one
		abstract := &NodeAbstractResource{Addr: addr}
		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}

		// Add it to the graph
		g.Add(node)
	}

	return nil
}

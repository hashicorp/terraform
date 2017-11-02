package terraform

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/terraform/config"
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

	// Module is the module to add resources from.
	Module *module.Tree

	// Unique will only add resources that aren't already present in the graph.
	Unique bool

	// Mode will only add resources that match the given mode
	ModeFilter bool
	Mode       config.ResourceMode

	l         sync.Mutex
	uniqueMap map[string]struct{}
}

func (t *ConfigTransformer) Transform(g *Graph) error {
	// Lock since we use some internal state
	t.l.Lock()
	defer t.l.Unlock()

	// If no module is given, we don't do anything
	if t.Module == nil {
		return nil
	}

	// If the module isn't loaded, that is simply an error
	if !t.Module.Loaded() {
		return errors.New("module must be loaded for ConfigTransformer")
	}

	// Reset the uniqueness map. If we're tracking uniques, then populate
	// it with addresses.
	t.uniqueMap = make(map[string]struct{})
	defer func() { t.uniqueMap = nil }()
	if t.Unique {
		for _, v := range g.Vertices() {
			if rn, ok := v.(GraphNodeResource); ok {
				t.uniqueMap[rn.ResourceAddr().String()] = struct{}{}
			}
		}
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
	conf := m.Config()

	// Build the path we're at
	path := m.Path()

	// Write all the resources out
	for _, r := range conf.Resources {
		// Build the resource address
		addr, err := parseResourceAddressConfig(r)
		if err != nil {
			panic(fmt.Sprintf(
				"Error parsing config address, this is a bug: %#v", r))
		}
		addr.Path = path

		// If this is already in our uniqueness map, don't add it again
		if _, ok := t.uniqueMap[addr.String()]; ok {
			continue
		}

		// Remove non-matching modes
		if t.ModeFilter && addr.Mode != t.Mode {
			continue
		}

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

type ProviderConfigTransformer struct {
	Providers []string
	Concrete  ConcreteProviderNodeFunc

	// Module is the module to add resources from.
	Module *module.Tree
}

func (t *ProviderConfigTransformer) Transform(g *Graph) error {
	// If no module is given, we don't do anything
	if t.Module == nil {
		return nil
	}

	// If the module isn't loaded, that is simply an error
	if !t.Module.Loaded() {
		return errors.New("module must be loaded for ProviderConfigTransformer")
	}

	// Start the transformation process
	return t.transform(g, t.Module)
}

func (t *ProviderConfigTransformer) transform(g *Graph, m *module.Tree) error {
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

func (t *ProviderConfigTransformer) transformSingle(g *Graph, m *module.Tree) error {
	log.Printf("[TRACE] ProviderConfigTransformer: Starting for path: %v", m.Path())

	// Get the configuration for this module
	conf := m.Config()

	// Build the path we're at
	path := m.Path()
	if len(path) > 0 {
		path = append([]string{RootModuleName}, path...)
	}

	// Write all the resources out
	for _, p := range conf.ProviderConfigs {
		name := p.Name
		if p.Alias != "" {
			name += "." + p.Alias
		}

		v := t.Concrete(&NodeAbstractProvider{
			NameValue: name,
			PathValue: path,
		}).(dag.Vertex)

		// Add it to the graph
		g.Add(v)
	}

	return nil
}

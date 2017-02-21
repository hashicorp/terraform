package terraform

import (
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// OrphanResourceTransformer is a GraphTransformer that adds resource
// orphans to the graph. A resource orphan is a resource that is
// represented in the state but not in the configuration.
//
// This only adds orphans that have no representation at all in the
// configuration.
type OrphanResourceTransformer struct {
	Concrete ConcreteResourceNodeFunc

	// State is the global state. We require the global state to
	// properly find module orphans at our path.
	State *State

	// Module is the root module. We'll look up the proper configuration
	// using the graph path.
	Module *module.Tree
}

func (t *OrphanResourceTransformer) Transform(g *Graph) error {
	if t.State == nil {
		// If the entire state is nil, there can't be any orphans
		return nil
	}

	// Go through the modules and for each module transform in order
	// to add the orphan.
	for _, ms := range t.State.Modules {
		if err := t.transform(g, ms); err != nil {
			return err
		}
	}

	return nil
}

func (t *OrphanResourceTransformer) transform(g *Graph, ms *ModuleState) error {
	if ms == nil {
		return nil
	}

	// Get the configuration for this path. The configuration might be
	// nil if the module was removed from the configuration. This is okay,
	// this just means that every resource is an orphan.
	var c *config.Config
	if m := t.Module.Child(ms.Path[1:]); m != nil {
		c = m.Config()
	}

	// Go through the orphans and add them all to the state
	for _, key := range ms.Orphans(c) {
		// Build the abstract resource
		addr, err := parseResourceAddressInternal(key)
		if err != nil {
			return err
		}
		addr.Path = ms.Path[1:]

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

package terraform

import (
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
)

// OrphanResourceTransformer is a GraphTransformer that adds resource
// orphans to the graph. A resource orphan is a resource that is
// represented in the state but not in the configuration.
//
// This only adds orphans that have no representation at all in the
// configuration.
type OrphanResourceTransformer struct {
	Concrete ConcreteResourceInstanceNodeFunc

	// State is the global state. We require the global state to
	// properly find module orphans at our path.
	State *State

	// Config is the root node in the configuration tree. We'll look up
	// the appropriate note in this tree using the path in each node.
	Config *configs.Config
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

	path := normalizeModulePath(ms.Path)

	// Get the configuration for this path. The configuration might be
	// nil if the module was removed from the configuration. This is okay,
	// this just means that every resource is an orphan.
	var m *configs.Module
	if c := t.Config.DescendentForInstance(path); c != nil {
		m = c.Module
	}

	// Go through the orphans and add them all to the state
	for _, relAddr := range ms.Orphans(m) {
		addr := relAddr.Absolute(path)
		abstract := NewNodeAbstractResourceInstance(addr)
		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}
		g.Add(node)
	}

	return nil
}

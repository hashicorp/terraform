package terraform

import (
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/states"
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
	State *states.State

	// Config is the root node in the configuration tree. We'll look up
	// the appropriate note in this tree using the path in each node.
	Config *configs.Config
}

func (t *OrphanResourceTransformer) Transform(g *Graph) error {
	if t.State == nil {
		// If the entire state is nil, there can't be any orphans
		return nil
	}
	if t.Config == nil {
		// Should never happen: we can't be doing any Terraform operations
		// without at least an empty configuration.
		panic("OrpahResourceTransformer used without setting Config")
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

func (t *OrphanResourceTransformer) transform(g *Graph, ms *states.Module) error {
	if ms == nil {
		return nil
	}

	moduleAddr := ms.Addr

	// Get the configuration for this module. The configuration might be
	// nil if the module was removed from the configuration. This is okay,
	// this just means that every resource is an orphan.
	var m *configs.Module
	if c := t.Config.DescendentForInstance(moduleAddr); c != nil {
		m = c.Module
	}

	// An "orphan" is a resource that is in the state but not the configuration,
	// so we'll walk the state resources and try to correlate each of them
	// with a configuration block. Each orphan gets a node in the graph whose
	// type is decided by t.Concrete.
	//
	// We don't handle orphans related to changes in the "count" and "for_each"
	// pseudo-arguments here. They are handled by OrphanResourceCountTransformer.
	for _, rs := range ms.Resources {
		if m != nil {
			if r := m.ResourceByAddr(rs.Addr); r != nil {
				continue
			}
		}

		for key := range rs.Instances {
			addr := rs.Addr.Instance(key).Absolute(moduleAddr)
			abstract := NewNodeAbstractResourceInstance(addr)
			var node dag.Vertex = abstract
			if f := t.Concrete; f != nil {
				node = f(abstract)
			}
			g.Add(node)
		}
	}

	return nil
}

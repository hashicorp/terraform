package terraform

import (
	"log"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/states"
)

// OrphanResourceInstanceTransformer is a GraphTransformer that adds orphaned
// resource instances to the graph. An "orphan" is an instance that is present
// in the state but belongs to a resource that is no longer present in the
// configuration.
//
// This is not the transformer that deals with "count orphans" (instances that
// are no longer covered by a resource's "count" or "for_each" setting); that's
// handled instead by OrphanResourceCountTransformer.
type OrphanResourceInstanceTransformer struct {
	Concrete ConcreteResourceInstanceNodeFunc

	// State is the global state. We require the global state to
	// properly find module orphans at our path.
	State *states.State

	// Config is the root node in the configuration tree. We'll look up
	// the appropriate note in this tree using the path in each node.
	Config *configs.Config
}

func (t *OrphanResourceInstanceTransformer) Transform(g *Graph) error {
	if t.State == nil {
		// If the entire state is nil, there can't be any orphans
		return nil
	}
	if t.Config == nil {
		// Should never happen: we can't be doing any Terraform operations
		// without at least an empty configuration.
		panic("OrphanResourceInstanceTransformer used without setting Config")
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

func (t *OrphanResourceInstanceTransformer) transform(g *Graph, ms *states.Module) error {
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
			log.Printf("[TRACE] OrphanResourceInstanceTransformer: adding single-instance orphan node for %s", addr)
			g.Add(node)
		}
	}

	return nil
}

// OrphanResourceTransformer is a GraphTransformer that adds orphaned
// resources to the graph. An "orphan" is a resource that is present in
// the state but no longer present in the config.
//
// This is separate to OrphanResourceInstanceTransformer in that it deals with
// whole resources, rather than individual instances of resources. Orphan
// resource nodes are only used during apply to clean up leftover empty
// resource state skeletons, after all of the instances inside have been
// removed.
//
// This transformer will also create edges in the graph to any pre-existing
// node that creates or destroys the entire orphaned resource or any of its
// instances, to ensure that the "orphan-ness" of a resource is always dealt
// with after all other aspects of it.
type OrphanResourceTransformer struct {
	Concrete ConcreteResourceNodeFunc

	// State is the global state.
	State *states.State

	// Config is the root node in the configuration tree.
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
		panic("OrphanResourceTransformer used without setting Config")
	}

	// We'll first collect up the existing nodes for each resource so we can
	// create dependency edges for any new nodes we create.
	deps := map[string][]dag.Vertex{}
	for _, v := range g.Vertices() {
		switch tv := v.(type) {
		case GraphNodeResourceInstance:
			k := tv.ResourceInstanceAddr().ContainingResource().String()
			deps[k] = append(deps[k], v)
		case GraphNodeResource:
			k := tv.ResourceAddr().String()
			deps[k] = append(deps[k], v)
		case GraphNodeDestroyer:
			k := tv.DestroyAddr().ContainingResource().String()
			deps[k] = append(deps[k], v)
		}
	}

	for _, ms := range t.State.Modules {
		moduleAddr := ms.Addr

		mc := t.Config.DescendentForInstance(moduleAddr) // might be nil if whole module has been removed

		for _, rs := range ms.Resources {
			if mc != nil {
				if r := mc.Module.ResourceByAddr(rs.Addr); r != nil {
					// It's in the config, so nothing to do for this one.
					continue
				}
			}

			addr := rs.Addr.Absolute(moduleAddr)
			abstract := NewNodeAbstractResource(addr)
			var node dag.Vertex = abstract
			if f := t.Concrete; f != nil {
				node = f(abstract)
			}
			log.Printf("[TRACE] OrphanResourceTransformer: adding whole-resource orphan node for %s", addr)
			g.Add(node)
			for _, dn := range deps[addr.String()] {
				log.Printf("[TRACE] OrphanResourceTransformer: node %q depends on %q", dag.VertexName(node), dag.VertexName(dn))
				g.Connect(dag.BasicEdge(node, dn))
			}
		}
	}

	return nil

}

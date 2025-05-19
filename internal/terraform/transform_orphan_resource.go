// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/states"
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

	// Do not apply this transformer
	skip bool
}

func (t *OrphanResourceInstanceTransformer) Transform(g *Graph) error {
	if t.skip {
		return nil
	}

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
	if c := t.Config.DescendantForInstance(moduleAddr); c != nil {
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
			if r := m.ResourceByAddr(rs.Addr.Resource); r != nil {
				continue
			}
		}

		for key, inst := range rs.Instances {
			// Instances which have no current objects (only one or more
			// deposed objects) will be taken care of separately
			if inst.Current == nil {
				continue
			}

			addr := rs.Addr.Instance(key)
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

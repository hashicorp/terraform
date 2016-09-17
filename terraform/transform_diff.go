package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/config/module"
)

// DiffTransformer is a GraphTransformer that adds the elements of
// the diff to the graph.
//
// This transform is used for example by the ApplyGraphBuilder to ensure
// that only resources that are being modified are represented in the graph.
//
// Module and State is still required for the DiffTransformer for annotations
// since the Diff doesn't contain all the information required to build the
// complete graph (such as create-before-destroy information). The graph
// is built based on the diff first, though, ensuring that only resources
// that are being modified are present in the graph.
type DiffTransformer struct {
	Diff   *Diff
	Module *module.Tree
	State  *State
}

func (t *DiffTransformer) Transform(g *Graph) error {
	// If the diff is nil or empty (nil is empty) then do nothing
	if t.Diff.Empty() {
		return nil
	}

	// Go through all the modules in the diff.
	log.Printf("[TRACE] DiffTransformer: starting")
	var nodes []*NodeApplyableResource
	for _, m := range t.Diff.Modules {
		log.Printf("[TRACE] DiffTransformer: Module: %s", m)
		// TODO: If this is a destroy diff then add a module destroy node

		// Go through all the resources in this module.
		for name, inst := range m.Resources {
			log.Printf("[TRACE] DiffTransformer: Resource %q: %#v", name, inst)

			// TODO: destroy
			if inst.Destroy {
			}

			// If this diff has no attribute changes, then we have
			// nothing to do and therefore won't add it to the graph.
			if len(inst.Attributes) == 0 {
				continue
			}

			// We have changes! This is a create or update operation.
			// First grab the address so we have a unique way to
			// reference this resource.
			addr, err := parseResourceAddressInternal(name)
			if err != nil {
				panic(fmt.Sprintf(
					"Error parsing internal name, this is a bug: %q", name))
			}

			// Very important: add the module path for this resource to
			// the address. Remove "root" from it.
			addr.Path = m.Path[1:]

			// Add the resource to the graph
			nodes = append(nodes, &NodeApplyableResource{
				Addr: addr,
			})
		}
	}

	// NOTE: Lots of room for performance optimizations below. For
	// resource-heavy diffs this part alone is probably pretty slow.

	// Annotate all nodes with their config and state
	for _, n := range nodes {
		// Grab the configuration at this path.
		if t := t.Module.Child(n.Addr.Path); t != nil {
			for _, r := range t.Config().Resources {
				// Get a resource address so we can compare
				addr, err := parseResourceAddressConfig(r)
				if err != nil {
					panic(fmt.Sprintf(
						"Error parsing config address, this is a bug: %#v", r))
				}
				addr.Path = n.Addr.Path

				// If this is not the same resource, then continue
				if !addr.Equals(n.Addr) {
					continue
				}

				// Same resource! Mark it and exit
				n.Config = r
				break
			}
		}

		// Grab the state at this path
		if ms := t.State.ModuleByPath(normalizeModulePath(n.Addr.Path)); ms != nil {
			for name, rs := range ms.Resources {
				// Parse the name for comparison
				addr, err := parseResourceAddressInternal(name)
				if err != nil {
					panic(fmt.Sprintf(
						"Error parsing internal name, this is a bug: %q", name))
				}
				addr.Path = n.Addr.Path

				// If this is not the same resource, then continue
				if !addr.Equals(n.Addr) {
					continue
				}

				// Same resource!
				n.ResourceState = rs
				break
			}
		}
	}

	// Add all the nodes to the graph
	for _, n := range nodes {
		g.Add(n)
	}

	return nil
}

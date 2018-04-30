package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/dag"
)

// DiffTransformer is a GraphTransformer that adds the elements of
// the diff to the graph.
//
// This transform is used for example by the ApplyGraphBuilder to ensure
// that only resources that are being modified are represented in the graph.
type DiffTransformer struct {
	Concrete ConcreteResourceInstanceNodeFunc
	Diff     *Diff
}

func (t *DiffTransformer) Transform(g *Graph) error {
	// If the diff is nil or empty (nil is empty) then do nothing
	if t.Diff.Empty() {
		return nil
	}

	// Go through all the modules in the diff.
	log.Printf("[TRACE] DiffTransformer: starting")
	var nodes []dag.Vertex
	for _, m := range t.Diff.Modules {
		log.Printf("[TRACE] DiffTransformer: Module: %s", m)
		// TODO: If this is a destroy diff then add a module destroy node

		// Go through all the resources in this module.
		for name, inst := range m.Resources {
			log.Printf("[TRACE] DiffTransformer: Resource %q: %#v", name, inst)

			// We have changes! This is a create or update operation.
			// First grab the address so we have a unique way to
			// reference this resource.
			legacyAddr, err := parseResourceAddressInternal(name)
			if err != nil {
				panic(fmt.Sprintf(
					"Error parsing internal name, this is a bug: %q", name))
			}
			addr := legacyAddr.AbsResourceInstanceAddr()

			// If we're destroying, add the destroy node
			if inst.Destroy || inst.GetDestroyDeposed() {
				abstract := NewNodeAbstractResourceInstance(addr)
				g.Add(&NodeDestroyResourceInstance{NodeAbstractResourceInstance: abstract})
			}

			// If we have changes, then add the applyable version
			if len(inst.Attributes) > 0 {
				// Add the resource to the graph
				abstract := NewNodeAbstractResourceInstance(addr)
				var node dag.Vertex = abstract
				if f := t.Concrete; f != nil {
					node = f(abstract)
				}

				nodes = append(nodes, node)
			}
		}
	}

	// Add all the nodes to the graph
	for _, n := range nodes {
		g.Add(n)
	}

	return nil
}

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/dag"
)

// StateTransformer is a GraphTransformer that adds the elements of
// the state to the graph.
//
// This transform is used for example by the DestroyPlanGraphBuilder to ensure
// that only resources that are in the state are represented in the graph.
type StateTransformer struct {
	Concrete ConcreteResourceInstanceNodeFunc

	State *State
}

func (t *StateTransformer) Transform(g *Graph) error {
	// If the state is nil or empty (nil is empty) then do nothing
	if t.State.Empty() {
		return nil
	}

	// Go through all the modules in the diff.
	log.Printf("[TRACE] StateTransformer: starting")
	var nodes []dag.Vertex
	for _, ms := range t.State.Modules {
		log.Printf("[TRACE] StateTransformer: Module: %v", ms.Path)

		// Go through all the resources in this module.
		for name, rs := range ms.Resources {
			log.Printf("[TRACE] StateTransformer: Resource %q: %#v", name, rs)

			// State hasn't yet been updated to our new address format, so
			// we need to shim this.
			legacyAddr, err := parseResourceAddressInternal(name)
			if err != nil {
				// Indicates someone has tampered with the state file
				return fmt.Errorf("invalid resource address %q in state", name)
			}
			// Very important: add the module path for this resource to
			// the address. Remove "root" from it.
			legacyAddr.Path = ms.Path[1:]

			addr := legacyAddr.AbsResourceInstanceAddr()

			// Add the resource to the graph
			abstract := NewNodeAbstractResourceInstance(addr)
			var node dag.Vertex = abstract
			if f := t.Concrete; f != nil {
				node = f(abstract)
			}

			nodes = append(nodes, node)
		}
	}

	// Add all the nodes to the graph
	for _, n := range nodes {
		g.Add(n)
	}

	return nil
}

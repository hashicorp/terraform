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
	Concrete ConcreteResourceNodeFunc

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

			// Add the resource to the graph
			addr, err := parseResourceAddressInternal(name)
			if err != nil {
				panic(fmt.Sprintf(
					"Error parsing internal name, this is a bug: %q", name))
			}

			// Very important: add the module path for this resource to
			// the address. Remove "root" from it.
			addr.Path = ms.Path[1:]

			// Add the resource to the graph
			abstract := &NodeAbstractResource{Addr: addr}
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

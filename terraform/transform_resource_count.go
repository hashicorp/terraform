package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/dag"
)

// ResourceCountTransformer is a GraphTransformer that expands the count
// out for a specific resource.
//
// This assumes that the count is already interpolated.
type ResourceCountTransformer struct {
	Concrete ConcreteResourceNodeFunc

	Count int
	Addr  *ResourceAddress
}

func (t *ResourceCountTransformer) Transform(g *Graph) error {
	// Don't allow the count to be negative
	if t.Count < 0 {
		return fmt.Errorf("negative count: %d", t.Count)
	}

	// For each count, build and add the node
	for i := 0; i < t.Count; i++ {
		// Set the index. If our count is 1 we special case it so that
		// we handle the "resource.0" and "resource" boundary properly.
		index := i
		if t.Count == 1 {
			index = -1
		}

		// Build the resource address
		addr := t.Addr.Copy()
		addr.Index = index

		// Build the abstract node and the concrete one
		abstract := &NodeAbstractResource{
			Addr: addr,
		}
		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}

		// Add it to the graph
		g.Add(node)
	}

	return nil
}

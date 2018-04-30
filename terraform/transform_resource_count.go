package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/dag"
)

// ResourceCountTransformer is a GraphTransformer that expands the count
// out for a specific resource.
//
// This assumes that the count is already interpolated.
type ResourceCountTransformer struct {
	Concrete ConcreteResourceInstanceNodeFunc

	// Count is either the number of indexed instances to create, or -1 to
	// indicate that count is not set at all and thus a no-key instance should
	// be created.
	Count int
	Addr  addrs.AbsResource
}

func (t *ResourceCountTransformer) Transform(g *Graph) error {
	if t.Count < 0 {
		// Negative count indicates that count is not set at all.
		addr := t.Addr.Instance(addrs.NoKey)

		abstract := NewNodeAbstractResourceInstance(addr)
		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}

		g.Add(node)
		return nil
	}

	// For each count, build and add the node
	for i := 0; i < t.Count; i++ {
		key := addrs.IntKey(i)
		addr := t.Addr.Instance(key)

		abstract := NewNodeAbstractResourceInstance(addr)
		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}

		g.Add(node)
	}

	return nil
}

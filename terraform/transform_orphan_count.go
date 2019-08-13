package terraform

import (
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/states"
	"github.com/zclconf/go-cty/cty"
)

// OrphanResourceCountTransformer is a GraphTransformer that adds orphans
// for an expanded count to the graph. The determination of this depends
// on the count argument given.
//
// Orphans are found by comparing the count to what is found in the state.
// This transform assumes that if an element in the state is within the count
// bounds given, that it is not an orphan.
type OrphanResourceCountTransformer struct {
	Concrete ConcreteResourceInstanceNodeFunc

	Count   int                  // Actual count of the resource, or -1 if count is not set at all
	ForEach map[string]cty.Value // The ForEach map on the resource
	Addr    addrs.AbsResource    // Addr of the resource to look for orphans
	State   *states.State        // Full global state
}

func (t *OrphanResourceCountTransformer) Transform(g *Graph) error {
	rs := t.State.Resource(t.Addr)
	if rs == nil {
		return nil // Resource doesn't exist in state, so nothing to do!
	}

	haveKeys := make(map[addrs.InstanceKey]struct{})
	for key := range rs.Instances {
		haveKeys[key] = struct{}{}
	}

	// if for_each is set, use that transformer
	if t.ForEach != nil {
		return t.transformForEach(haveKeys, g)
	}
	if t.Count < 0 {
		return t.transformNoCount(haveKeys, g)
	}
	if t.Count == 0 {
		return t.transformZeroCount(haveKeys, g)
	}
	return t.transformCount(haveKeys, g)
}

func (t *OrphanResourceCountTransformer) transformForEach(haveKeys map[addrs.InstanceKey]struct{}, g *Graph) error {
	// If there is a no-key node, add this to the graph first,
	// so that we can create edges to it in subsequent (StringKey) nodes.
	// This is because the last item determines the resource mode for the whole resource,
	// so if this (non-deterministically) happens to end up as the last one,
	// that will change the resource's EachMode and our addressing for our instances
	// will not work as expected
	_, hasNoKeyNode := haveKeys[addrs.NoKey]
	var noKeyNode dag.Vertex
	if hasNoKeyNode {
		abstract := NewNodeAbstractResourceInstance(t.Addr.Instance(addrs.NoKey))
		noKeyNode = abstract
		if f := t.Concrete; f != nil {
			noKeyNode = f(abstract)
		}
		g.Add(noKeyNode)
	}

	for key := range haveKeys {
		s, _ := key.(addrs.StringKey)
		// If the key is present in our current for_each, carry on
		if _, ok := t.ForEach[string(s)]; ok {
			continue
		}

		// If the key is no-key, we have already added it, so skip
		if key == addrs.NoKey {
			continue
		}

		abstract := NewNodeAbstractResourceInstance(t.Addr.Instance(key))
		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}
		log.Printf("[TRACE] OrphanResourceCount(non-zero): adding %s as %T", t.Addr, node)
		g.Add(node)

		// Add edge to noKeyNode if it exists
		if hasNoKeyNode {
			g.Connect(dag.BasicEdge(node, noKeyNode))
		}
	}
	return nil
}

func (t *OrphanResourceCountTransformer) transformCount(haveKeys map[addrs.InstanceKey]struct{}, g *Graph) error {
	// Due to the logic in Transform, we only get in here if our count is
	// at least one.

	_, have0Key := haveKeys[addrs.IntKey(0)]

	for key := range haveKeys {
		if key == addrs.NoKey && !have0Key {
			// If we have no 0-key then we will accept a no-key instance
			// as an alias for it.
			continue
		}

		i, isInt := key.(addrs.IntKey)
		if isInt && int(i) < t.Count {
			continue
		}

		abstract := NewNodeAbstractResourceInstance(t.Addr.Instance(key))
		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}
		log.Printf("[TRACE] OrphanResourceCount(non-zero): adding %s as %T", t.Addr, node)
		g.Add(node)
	}

	return nil
}

func (t *OrphanResourceCountTransformer) transformZeroCount(haveKeys map[addrs.InstanceKey]struct{}, g *Graph) error {
	// This case is easy: we need to orphan any keys we have at all.

	for key := range haveKeys {
		abstract := NewNodeAbstractResourceInstance(t.Addr.Instance(key))
		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}
		log.Printf("[TRACE] OrphanResourceCount(zero): adding %s as %T", t.Addr, node)
		g.Add(node)
	}

	return nil
}

func (t *OrphanResourceCountTransformer) transformNoCount(haveKeys map[addrs.InstanceKey]struct{}, g *Graph) error {
	// Negative count indicates that count is not set at all, in which
	// case we expect to have a single instance with no key set at all.
	// However, we'll also accept an instance with key 0 set as an alias
	// for it, in case the user has just deleted the "count" argument and
	// so wants to keep the first instance in the set.

	_, haveNoKey := haveKeys[addrs.NoKey]
	_, have0Key := haveKeys[addrs.IntKey(0)]
	keepKey := addrs.NoKey
	if have0Key && !haveNoKey {
		// If we don't have a no-key instance then we can use the 0-key instance
		// instead.
		keepKey = addrs.IntKey(0)
	}

	for key := range haveKeys {
		if key == keepKey {
			continue
		}

		abstract := NewNodeAbstractResourceInstance(t.Addr.Instance(key))
		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}
		log.Printf("[TRACE] OrphanResourceCount(no-count): adding %s as %T", t.Addr, node)
		g.Add(node)
	}

	return nil
}

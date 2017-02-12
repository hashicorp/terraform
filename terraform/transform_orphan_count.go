package terraform

import (
	"log"

	"github.com/hashicorp/terraform/dag"
)

// OrphanResourceCountTransformer is a GraphTransformer that adds orphans
// for an expanded count to the graph. The determination of this depends
// on the count argument given.
//
// Orphans are found by comparing the count to what is found in the state.
// This transform assumes that if an element in the state is within the count
// bounds given, that it is not an orphan.
type OrphanResourceCountTransformer struct {
	Concrete ConcreteResourceNodeFunc

	Count int              // Actual count of the resource
	Addr  *ResourceAddress // Addr of the resource to look for orphans
	State *State           // Full global state
}

func (t *OrphanResourceCountTransformer) Transform(g *Graph) error {
	log.Printf("[TRACE] OrphanResourceCount: Starting...")

	// Grab the module in the state just for this resource address
	ms := t.State.ModuleByPath(normalizeModulePath(t.Addr.Path))
	if ms == nil {
		// If no state, there can't be orphans
		return nil
	}

	orphanIndex := -1
	if t.Count == 1 {
		orphanIndex = 0
	}

	// Go through the orphans and add them all to the state
	for key, _ := range ms.Resources {
		// Build the address
		addr, err := parseResourceAddressInternal(key)
		if err != nil {
			return err
		}
		addr.Path = ms.Path[1:]

		// Copy the address for comparison. If we aren't looking at
		// the same resource, then just ignore it.
		addrCopy := addr.Copy()
		addrCopy.Index = -1
		if !addrCopy.Equals(t.Addr) {
			continue
		}

		log.Printf("[TRACE] OrphanResourceCount: Checking: %s", addr)

		idx := addr.Index

		// If we have zero and the index here is 0 or 1, then we
		// change the index to a high number so that we treat it as
		// an orphan.
		if t.Count <= 0 && idx <= 0 {
			idx = t.Count + 1
		}

		// If we have a count greater than 0 and we're at the zero index,
		// we do a special case check to see if our state also has a
		// -1 index value. If so, this is an orphan because our rules are
		// that if both a -1 and 0 are in the state, the 0 is destroyed.
		if t.Count > 0 && idx == orphanIndex {
			// This is a piece of cleverness (beware), but its simple:
			// if orphanIndex is 0, then check -1, else check 0.
			checkIndex := (orphanIndex + 1) * -1

			key := &ResourceStateKey{
				Name:  addr.Name,
				Type:  addr.Type,
				Mode:  addr.Mode,
				Index: checkIndex,
			}

			if _, ok := ms.Resources[key.String()]; ok {
				// We have a -1 index, too. Make an arbitrarily high
				// index so that we always mark this as an orphan.
				log.Printf(
					"[WARN] OrphanResourceCount: %q both -1 and 0 index found, orphaning %d",
					addr, orphanIndex)
				idx = t.Count + 1
			}
		}

		// If the index is within the count bounds, it is not an orphan
		if idx < t.Count {
			continue
		}

		// Build the abstract node and the concrete one
		abstract := &NodeAbstractResource{Addr: addr}
		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}

		// Add it to the graph
		g.Add(node)
	}

	return nil
}

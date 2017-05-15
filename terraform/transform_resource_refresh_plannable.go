package terraform

import (
	"fmt"
	"log"
)

// ResourceRefreshPlannableTransformer is a GraphTransformer that replaces any
// nodes that don't have state yet exist in config with
// NodePlannableResourceInstance.
//
// This transformer is used when expanding count on managed resource nodes
// during the refresh phase to ensure that data sources that have
// interpolations that depend on resources existing in the graph can be walked
// properly.
type ResourceRefreshPlannableTransformer struct {
	// The full global state.
	State *State
}

// Transform implements GraphTransformer for
// ResourceRefreshPlannableTransformer.
func (t *ResourceRefreshPlannableTransformer) Transform(g *Graph) error {
nextVertex:
	for _, v := range g.Vertices() {
		addr := v.(*NodeRefreshableManagedResourceInstance).Addr

		// Find the state for this address, if there is one
		filter := &StateFilter{State: t.State}
		results, err := filter.Filter(addr.String())
		if err != nil {
			return err
		}

		// Check to see if we have a state for this resource. If we do, skip this
		// node.
		for _, result := range results {
			if _, ok := result.Value.(*ResourceState); ok {
				continue nextVertex
			}
		}
		// If we don't, convert this resource to a NodePlannableResourceInstance node
		// with all of the data we need to make it happen.
		log.Printf("[TRACE] No state for %s, converting to NodePlannableResourceInstance", addr.String())
		new := &NodePlannableResourceInstance{
			NodeAbstractResource: v.(*NodeRefreshableManagedResourceInstance).NodeAbstractResource,
		}
		// Replace the node in the graph
		if !g.Replace(v, new) {
			return fmt.Errorf("ResourceRefreshPlannableTransformer: Could not replace node %#v with %#v", v, new)
		}
	}

	return nil
}

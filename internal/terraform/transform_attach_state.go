// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
)

// GraphNodeAttachResourceInstanceState is an interface that can be implemented
// to request that a ResourceState is attached to the node.
//
// Due to a historical naming inconsistency, the type ResourceState actually
// represents the state for a particular _instance_, while InstanceState
// represents the values for that instance during a particular phase
// (e.g. primary vs. deposed). Consequently, GraphNodeAttachResourceInstanceState
// is supported only for nodes that represent resource instances, even though
// the name might suggest it is for containing resources.
type GraphNodeAttachResourceInstanceState interface {
	GraphNodeResourceInstance

	// Sets the state
	AttachResourceInstanceState(*states.ResourceInstance, addrs.AbsProviderConfig)
}

// AttachStateTransformer goes through the graph and attaches
// state to nodes that implement the interfaces above.
type AttachStateTransformer struct {
	State *states.State // State is the root state
}

func (t *AttachStateTransformer) Transform(g *Graph) error {
	// If no state, then nothing to do
	if t.State == nil {
		log.Printf("[DEBUG] Not attaching any node states: overall state is nil")
		return nil
	}

	for v := range g.VerticesSeq() {
		// Nodes implement this interface to request state attachment.
		an, ok := v.(GraphNodeAttachResourceInstanceState)
		if !ok {
			continue
		}
		addr := an.ResourceInstanceAddr()

		rs := t.State.Resource(addr.ContainingResource())
		if rs == nil {
			log.Printf("[DEBUG] Resource state not found for node %q, instance %s", v.Name(), addr)
			continue
		}

		is := rs.Instance(addr.Resource.Key)
		if is == nil {
			// It'd be weird for the specific instance we're attaching to not
			// to exist.
			log.Printf("[DEBUG] Resource instance state not found for node %q, instance %s", v.Name(), addr)
			continue
		}

		// Attach a copy containing only this node's own instance, so that
		// instances can't modify each other's state. Copying the entire
		// resource for every instance would be quadratic in the number of
		// instances, and nodes only retain their own instance anyway.
		an.AttachResourceInstanceState(is.DeepCopy(), rs.ProviderConfig)
	}

	return nil
}

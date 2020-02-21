package terraform

import (
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/states"
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

	Addr          addrs.AbsResource           // Addr of the resource to look for orphans
	InstanceAddrs []addrs.AbsResourceInstance // Addresses that currently exist in config
	State         *states.State               // Full global state
}

func (t *OrphanResourceCountTransformer) Transform(g *Graph) error {
	// FIXME: This is currently assuming that all of the instances of
	// this resource belong to a single module instance, which is true
	// at the time of writing this because Terraform Core doesn't support
	// repetition of module calls yet, but this will need to be corrected
	// in order to support count and for_each on module calls, where
	// our t.InstanceAddrs may contain resource instances from many different
	// module instances.
	rs := t.State.Resource(t.Addr)
	if rs == nil {
		return nil // Resource doesn't exist in state, so nothing to do!
	}

	// This is an O(n*m) analysis, which we accept for now because the
	// number of instances of a single resource ought to always be small in any
	// reasonable Terraform configuration.
Have:
	for key := range rs.Instances {
		thisAddr := t.Addr.Instance(key)
		for _, wantAddr := range t.InstanceAddrs {
			if wantAddr.Equal(thisAddr) {
				continue Have
			}
		}
		// If thisAddr is not in t.InstanceAddrs then we've found an "orphan"

		abstract := NewNodeAbstractResourceInstance(thisAddr)
		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}
		log.Printf("[TRACE] OrphanResourceCountTransformer: adding %s as %T", thisAddr, node)
		g.Add(node)
	}

	return nil
}

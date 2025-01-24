// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
)

type GraphNodeLockable interface {
	// AttachSemaphore attaches a semaphore to the node.
	AttachSemaphore(Semaphore)

	// Lock locks the node.
	Lock()

	// Unlock unlocks the node.
	Unlock()
}

// DynamicConcurrencyTransformer is a GraphTransformer that attaches semaphores to
// resource instances whose configuration specifies a concurrency limit.
type DynamicConcurrencyTransformer struct {
	State *states.State
}

func (t *DynamicConcurrencyTransformer) Transform(g *Graph) error {
	log.Printf("[TRACE] DynamicConcurrencyTransformer starting")

	for _, node := range g.Vertices() {
		rn, ok := node.(*nodeExpandApplyableResource)
		if !ok {
			continue
		}

		// If the concurrency is not configured, we don't need to do anything.
		if rn.Config == nil || rn.Config.Managed == nil || rn.Config.Managed.Concurrency < 1 {
			continue
		}

		// Get all instances of the resource node, and attach the semaphore to them.
		descendants := g.Descendants(rn)
		for _, n := range descendants {
			if n, ok := n.(GraphNodeResourceInstance); ok && n.ResourceInstanceAddr().ConfigResource().Equal(rn.Addr) {
				if n, ok := n.(GraphNodeLockable); ok {
					n.AttachSemaphore(rn.getSemaphore())
				}
			}
		}
	}

	// We'll go through all the destroy nodes and attach semaphores to them.
	// Because destroy nodes may not have a corresponding configuration, we'll
	// need to look at the state to determine the concurrency limit.
	concurrencies := addrs.MakeMap[addrs.Resource, int]()
	semaphores := addrs.MakeMap[addrs.Resource, Semaphore]()
	resourceNodes := addrs.MakeMap[addrs.Resource, []*NodeDestroyResourceInstance]()
	for _, node := range g.Vertices() {
		rn, ok := node.(*NodeDestroyResourceInstance)
		if !ok {
			continue
		}
		resource := rn.ResourceAddr().Resource

		// If the destroy node already has a semaphore attached from the
		// previous step, we don't need to do anything.
		// We would however store the semaphore in the semaphores map, and
		// use it for the node's siblings that don't have a semaphore attached.
		if rn.semaphore != nil {
			semaphores.Put(resource, rn.semaphore)
			continue
		}

		// No semaphore yet, which means the node doesn't have a corresponding
		// configuration. We'll need to look at the state to determine the
		// concurrency limit.
		inst := t.State.ResourceInstance(rn.ResourceInstanceAddr())

		// If the resource instance is not in the state, or the concurrency is not set,
		// we don't need to do anything. This just means that the instances
		// will not be locked to the resource's concurrency limit.
		if !inst.HasCurrent() || inst.Current.Concurrency < 1 {
			continue
		}
		if concurrencies.Has(resource) && concurrencies.Get(resource) != inst.Current.Concurrency {
			log.Printf("[WARN] DynamicConcurrencyTransformer: resource %s has conflicting concurrency settings in state", resource)
		}
		concurrencies.Put(resource, inst.Current.Concurrency)
		resourceNodes.Put(resource, append(resourceNodes.Get(resource), rn))
	}

	for _, conc := range concurrencies.Elems {
		nodes := resourceNodes.Get(conc.Key)
		sem := semaphores.Get(conc.Key)
		if sem == nil {
			sem = NewSemaphore(conc.Value)
		}
		for _, n := range nodes {
			n.AttachSemaphore(sem)
		}
	}

	log.Printf("[TRACE] DynamicConcurrencyTransformer complete")
	return nil
}

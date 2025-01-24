// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"
)

type GraphNodeAttachSemaphore interface {
	// AttachSemaphore attaches a semaphore to the node.
	AttachSemaphore(Semaphore)
}

type GraphNodeLockable interface {
	GraphNodeAttachSemaphore
	// Lock locks the node.
	Lock()
	// Unlock unlocks the node.
	Unlock()
}

// DynamicConcurrencyTransformer is a GraphTransformer that attaches semaphores to
// resource instances whose configuration specifies a concurrency limit.
type DynamicConcurrencyTransformer struct {
}

func (t *DynamicConcurrencyTransformer) Transform(g *Graph) error {
	log.Printf("[TRACE] DynamicConcurrencyTransformer starting")

	for _, node := range g.Vertices() {
		rn, ok := node.(*nodeExpandApplyableResource)
		if !ok {
			continue
		}

		// If the concurrency is not configures, we don't need to do anything.
		if rn.Config == nil || rn.Config.Managed == nil || rn.Config.Managed.Concurrency < 1 {
			continue
		}

		// Get all instances of the resource node. We remove them from the graph, and
		// add them to the resource node. The resource node will be responsible for
		// executing the instances.
		descendants := g.Descendants(rn)
		for _, n := range descendants {
			if n, ok := n.(GraphNodeResourceInstance); ok && n.ResourceInstanceAddr().ConfigResource().Equal(rn.Addr) {
				if n, ok := n.(GraphNodeAttachSemaphore); ok {
					n.AttachSemaphore(rn.getSemaphore())
				}
			}
		}
	}

	log.Printf("[TRACE] DynamicConcurrencyTransformer complete")
	return nil
}

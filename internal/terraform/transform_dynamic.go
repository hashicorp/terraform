// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
)

type GraphNodeLockable interface {
	GraphNodeLifecycle
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

	semaphores := addrs.MakeMap[addrs.Resource, Semaphore]()

	for _, node := range g.Vertices() {
		n, ok := node.(GraphNodeConfigResource)
		if !ok {
			continue
		}

		lockable, ok := node.(GraphNodeLockable)
		if !ok {
			continue
		}

		concurrency := lockable.Concurrency()
		if concurrency < 1 {
			continue
		}

		sem := semaphores.Get(n.ResourceAddr().Resource)
		if sem == nil {
			sem = NewSemaphore(concurrency)
			semaphores.Put(n.ResourceAddr().Resource, sem)
		}
		lockable.AttachSemaphore(sem)
	}

	for _, node := range g.Vertices() {
		n, ok := node.(GraphNodeConfigResource)
		if !ok {
			continue
		}

		sem := semaphores.Get(n.ResourceAddr().Resource)
		if n, ok := node.(GraphNodeLockable); ok && sem != nil {
			n.AttachSemaphore(sem)
		}

	}

	log.Printf("[TRACE] DynamicConcurrencyTransformer complete")
	return nil
}

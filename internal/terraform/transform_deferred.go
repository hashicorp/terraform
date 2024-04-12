// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
)

// DeferredTransformer is a GraphTransformer that adds graph nodes representing
// each of the deferred changes to the graph.
//
// Deferred changes are not executed during the apply phase, but they are
// tracked in the graph to ensure that the correct ordering is maintained and
// the target flags are correctly applied.
type DeferredTransformer struct {
	DeferredChanges []*plans.DeferredResourceInstanceChangeSrc
}

func (t *DeferredTransformer) Transform(g *Graph) error {
	if t.DeferredChanges == nil || len(t.DeferredChanges) == 0 {
		return nil
	}

	for _, change := range t.DeferredChanges {
		node := &nodeApplyableDeferredInstance{
			NodeAbstractResourceInstance: NewNodeAbstractResourceInstance(change.ChangeSrc.Addr),
		}

		// Create a special node for partial instances, that handles the
		// addresses a little differently.
		if change.DeferredReason == providers.DeferredReasonInstanceCountUnknown {
			per := change.ChangeSrc.Addr.PartialResource()

			// This is a partial instance, so we need to create a partial node
			// instead of a full instance node.
			g.Add(&nodeApplyableDeferredPartialInstance{
				nodeApplyableDeferredInstance: node,
				PartialAddr:                   per,
			})

			// Now we want to find the expansion node that would be applied for
			// this resource, and tell it that it is performing a partial
			// expansion.
			for _, v := range g.Vertices() {
				if n, ok := v.(*nodeExpandApplyableResource); ok {
					if per.ConfigResource().Equal(n.Addr) {
						n.PartialExpansions = append(n.PartialExpansions, per)
					}
				}
			}

			continue
		}

		// Otherwise, just add the normal deferred instance node.
		g.Add(node)
	}

	return nil
}

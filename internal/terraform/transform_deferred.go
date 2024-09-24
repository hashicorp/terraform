// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
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
	if len(t.DeferredChanges) == 0 {
		return nil
	}

	// As with the DiffTransformer, DeferredTransformer creates resource
	// instance nodes. If there are any whole-resource nodes already in the
	// graph, we must ensure they get evaluated before any of the corresponding
	// instances by creating dependency edges.
	resourceNodes := addrs.MakeMap[addrs.ConfigResource, []GraphNodeConfigResource]()
	for _, node := range g.Vertices() {
		rn, ok := node.(GraphNodeConfigResource)
		if !ok {
			continue
		}
		// We ignore any instances that _also_ implement
		// GraphNodeResourceInstance, since in the unlikely event that they
		// do exist we'd probably end up creating cycles by connecting them.
		if _, ok := node.(GraphNodeResourceInstance); ok {
			continue
		}

		rAddr := rn.ResourceAddr()
		resourceNodes.Put(rAddr, append(resourceNodes.Get(rAddr), rn))
	}

	for _, change := range t.DeferredChanges {
		node := &nodeApplyableDeferredInstance{
			NodeAbstractResourceInstance: NewNodeAbstractResourceInstance(change.ChangeSrc.Addr),
			Reason:                       change.DeferredReason,
			ChangeSrc:                    change.ChangeSrc,
		}

		// Create a special node for partial instances, that handles the
		// addresses a little differently.
		if change.DeferredReason == providers.DeferredReasonInstanceCountUnknown {
			per := change.ChangeSrc.Addr.PartialResource()

			// This is a partial instance, so we need to create a partial node
			// instead of a full instance node.
			node := &nodeApplyableDeferredPartialInstance{
				nodeApplyableDeferredInstance: node,
				PartialAddr:                   per,
			}
			g.Add(node)

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

			// Also connect the deferred instance node to the underlying
			// resource node to make sure any expansion happens first.
			for _, resourceNode := range resourceNodes.Get(node.Addr.ConfigResource()) {
				g.Connect(dag.BasicEdge(node, resourceNode))
			}

			continue
		}

		// Otherwise, just add the normal deferred instance node.
		g.Add(node)

		// Still connect the deferred instance node to the underlying resource
		// node.
		for _, resourceNode := range resourceNodes.Get(node.Addr.ConfigResource()) {
			g.Connect(dag.BasicEdge(node, resourceNode))
		}
	}

	return nil
}

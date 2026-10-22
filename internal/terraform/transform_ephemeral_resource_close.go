// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang"
)

// ephemeralResourceCloseTransformer is a graph transformer that inserts a
// nodeEphemeralResourceClose node for each ephemeral resource, and arranges for
// the close node to depend on any other node that consumes the relevant
// ephemeral resource.
//
// Normally each close node runs as soon as the last consumer of its ephemeral
// resource has finished. If a hold callback is provided, every close node
// additionally depends on a [nodeEphemeralHold] node, which in turn depends on all of
// the other work in the graph. All ephemeral resources are then kept open (and
// renewed) until the whole walk has completed and the caller has released the hold.
type ephemeralResourceCloseTransformer struct {
	// This does not need to run during validate walks since the ephemeral
	// resources will never be opened.
	skip bool

	// hold is a callback function that, when provided, informs this transformer to
	// add the [nodeEphemeralHold] node to the graph. This node calls the callback function
	// with the root module scope and guarantees that the ephemeral data will be kept valid until
	// the release channel is closed.
	//
	// When nil, no hold node is added and each close node only depends on the
	// consumers of its own ephemeral resource.
	hold func(scope *lang.Scope) <-chan struct{}
}

func (t *ephemeralResourceCloseTransformer) Transform(g *Graph) error {
	if t.skip {
		// Nothing to do if ephemeral resources are not opened
		return nil
	}

	// If provided, add the hold node to the graph before adding the close nodes
	hold := t.addHold(g)

	verts := g.VerticesSeq()
	for v := range verts {
		// find any ephemeral resource nodes
		v, ok := v.(GraphNodeConfigResource)
		if !ok {
			continue
		}
		addr := v.ResourceAddr()
		if addr.Resource.Mode != addrs.EphemeralResourceMode {
			continue
		}

		closeNode := &nodeEphemeralResourceClose{
			// the node must also be a ProviderConsumer
			resourceNode: v.(GraphNodeProviderConsumer),
			addr:         addr,
		}
		log.Printf("[TRACE] ephemeralResourceCloseTransformer: adding close node for %s", addr)
		g.Add(closeNode)
		g.Connect(closeNode, v)
		if hold != nil {
			g.Connect(closeNode, hold)
		}

		// Now we have an ephemeral resource, and we need to depend on all
		// dependents of that resource. Rather than connect directly to them all
		// however, we'll only connect to leaf nodes by finding those that have
		// no up edges.
		lastReferences := g.FirstDescendantsWith(v, func(v dag.Vertex) bool {
			// We want something which is both a referencer and has no incoming
			// edges from referencers. While it wouldn't be incorrect to just
			// check for all leaf nodes, we are trying to connect to the end of
			// evaluation chain, otherwise we may just as well wait until the end
			// of the walk and close everything together. We technically don't
			// know if these nodes are connected because they reference the
			// ephemeral value, or if they are connected for some other
			// dependency reason, but this generally shouldn't matter as we can
			// count any dependency as a reason to maintain the ephemeral value.
			if _, ok := v.(GraphNodeReferencer); !ok {
				return false
			}

			up := g.EdgesTo(v)
			up = up.Filter(func(v dag.Vertex) bool {
				_, ok := v.(GraphNodeReferencer)
				return ok
			})

			// if there are no references connected to this node, then we can be
			// sure it's the last referencer in the chain.
			return up.Len() == 0
		})

		for last := range lastReferences.All() {
			g.Connect(closeNode, last)
		}
	}
	return nil
}

// addHold adds the hold node, if requested, and returns it. The node depends on
// every node currently in the graph that has no dependents, so it is only
// reached when the rest of the walk completed without errors.
func (t *ephemeralResourceCloseTransformer) addHold(g *Graph) *nodeEphemeralHold {
	if t.hold == nil {
		return nil
	}

	hold := &nodeEphemeralHold{callback: t.hold}
	g.Add(hold)

	for v := range g.VerticesSeq() {
		if v == hold {
			continue
		}

		if g.EdgesTo(v).Len() == 0 {
			g.Connect(hold, v)
		}
	}

	return hold
}

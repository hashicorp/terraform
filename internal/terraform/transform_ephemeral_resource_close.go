// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
)

// ephemeralResourceCloseTransformer is a graph transformer that inserts a
// nodeEphemeralResourceClose node for each ephemeral resource, and arranges for
// the close node to depend on any other node that consumes the relevant
// ephemeral resource.
type ephemeralResourceCloseTransformer struct {
	// This does not need to run during validate walks since the ephemeral
	// resources will never be opened.
	skip bool
}

func (t *ephemeralResourceCloseTransformer) Transform(g *Graph) error {
	if t.skip {
		// Nothing to do if ephemeral resources are not opened
		return nil
	}

	verts := g.Vertices()
	for _, v := range verts {
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
		g.Connect(dag.BasicEdge(closeNode, v))

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

			up := g.UpEdges(v)
			up = up.Filter(func(v any) bool {
				_, ok := v.(GraphNodeReferencer)
				return ok
			})

			// if there are no references connected to this node, then we can be
			// sure it's the last referencer in the chain.
			return len(up) == 0
		})

		for _, last := range lastReferences.List() {
			g.Connect(dag.BasicEdge(closeNode, last))
		}
	}
	return nil
}

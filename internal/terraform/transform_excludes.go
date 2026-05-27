// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
)

// ExcludesTransformer is a GraphTransformer that, when the user specifies a
// list of resources to exclude, limits the graph to everything except those
// resources and anything dependent on those resources.
type ExcludesTransformer struct {
	// List of excluded resource names specified by the user.
	Excludes []addrs.Targetable
}

func (t *ExcludesTransformer) Transform(g *Graph) error {
	if len(t.Excludes) == 0 {
		return nil
	}

	excludedNodes := t.selectExcludedNodes(g, t.Excludes)
	for _, v := range g.Vertices() {
		if excludedNodes.Include(v) {
			log.Printf("[DEBUG] Removing %q, filtered by targeting (excluded).", dag.VertexName(v))
			g.Remove(v)
		}
	}

	return nil
}

func (t *ExcludesTransformer) selectExcludedNodes(g *Graph, addrs []addrs.Targetable) dag.Set {
	excludedNodes := make(dag.Set)
	if len(addrs) == 0 {
		return excludedNodes
	}

	vertices := g.Vertices()

	for _, v := range vertices {
		if t.nodeIsExcluded(v, addrs) {
			// Add node and any descendants to excludedNodes
			t.addVertexDependenciesToExcludedNodes(g, v, excludedNodes, addrs)

			// We inform nodes that ask about the list of excludes - helps for nodes
			// that need to dynamically expand. Note that this only occurs for nodes
			// that are already directly excluded.
			if tn, ok := v.(GraphNodeTargetable); ok {
				tn.SetExcludes(addrs)
			}
		}
	}

	return excludedNodes
}

func (t *ExcludesTransformer) nodeIsExcluded(v dag.Vertex, excludes []addrs.Targetable) bool {
	var vertexAddr addrs.Targetable
	switch r := v.(type) {
	case *nodeApplyableDeferredPartialInstance:
		// TODO: Should verify that this comment is true + that we don't need to implement anything further for deferred changes.
		//
		// We can't exclude partial nodes as we don't have enough information to be certain that they should be excluded.
		return false

	case GraphNodeResourceInstance:
		vertexAddr = r.ResourceInstanceAddr()
	case GraphNodeConfigResource:
		vertexAddr = r.ResourceAddr()

	default:
		// Only resource and resource instance nodes can be excluded.
		return false
	}

	for _, excludeAddr := range excludes {
		// In the case of an absolute instance, we cannot exclude the node (or it's dependants) until expansion has occurred,
		// so we cannot generalize the excludeAddr like TargetTransformer does.
		if excludeAddr.TargetContains(vertexAddr) {
			return true
		}
	}

	return false
}

// addVertexDependenciesToExcludedNodes adds dependencies of the excluded vertex to the
// excludedNodes set. This includes all descendants in the graph.
func (t *ExcludesTransformer) addVertexDependenciesToExcludedNodes(g *Graph, v dag.Vertex, excludedNodes dag.Set, addrs []addrs.Targetable) {
	if excludedNodes.Include(v) {
		return
	}
	excludedNodes.Add(v)

	// TODO: Consider nodes that could appear as descendants that we don't want to exclude, for example: nodeCloseModule
	for _, d := range g.Descendants(v) {
		t.addVertexDependenciesToExcludedNodes(g, d, excludedNodes, addrs)
	}
}

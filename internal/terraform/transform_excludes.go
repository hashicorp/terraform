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
	// List of excluded resource addresses specified by the user.
	Excludes []addrs.Targetable
}

func (t *ExcludesTransformer) Transform(g *Graph) error {
	if len(t.Excludes) == 0 {
		return nil
	}

	excludedNodes := t.selectExcludedNodes(g, t.Excludes)
	for v := range g.VerticesSeq() {
		if excludedNodes.Contains(v) {
			log.Printf("[DEBUG] Removing %q, filtered by targeting (excluded).", v.Name())
			g.Remove(v)
		}
	}

	return nil
}

func (t *ExcludesTransformer) selectExcludedNodes(g *Graph, addrs []addrs.Targetable) dag.VertexSet {
	excludedNodes := dag.NewVertexSet()
	if len(addrs) == 0 {
		return excludedNodes
	}

	vertices := g.VerticesSeq()

	for v := range vertices {
		if t.nodeIsExcluded(v, addrs) {
			// Add node and any descendants to excludedNodes
			t.addVertexDependenciesToExcludedNodes(g, v, excludedNodes, addrs)
		}
	}

	return excludedNodes
}

func (t *ExcludesTransformer) nodeIsExcluded(v dag.Vertex, excludes []addrs.Targetable) bool {
	var vertexAddr addrs.Targetable
	switch r := v.(type) {
	case *nodeApplyableDeferredPartialInstance:
		// TODO:@austinvalle: Should verify that this comment is true + that we don't need to implement anything further
		// for deferred changes. We can't exclude partial nodes as we don't have enough information to be certain that
		// they should be excluded.
		//
		// Regardless, I should write a test for this.
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
		if excludeAddr.TargetContains(vertexAddr) {
			return true
		}

		// When excluding an expanded instance, we cannot exclude the node (or it's dependants)
		// until expansion has occurred.
		//
		// If the vertex address is contained in the generalized exclude address, then we
		// inform the node of the exclude addresses which will be processed in the ExcludesTransformer
		// in the resource's subgraph.
		//
		// TODO:@austinvalle: Should I just be looking for nodePlannableResource?
		if _, ok := vertexAddr.(addrs.ConfigResource); ok {
			switch exclude := excludeAddr.(type) {
			case addrs.AbsResourceInstance:
				excludeAddr = exclude.ContainingResource().Config()
			case addrs.AbsResource:
				excludeAddr = exclude.Config()
			case addrs.ModuleInstance:
				excludeAddr = exclude.Module()
			}

			if excludeAddr.TargetContains(vertexAddr) {
				if tn, ok := v.(GraphNodeTargetable); ok {
					tn.SetExcludes(excludes)
				}
			}
		}
	}

	return false
}

// addVertexDependenciesToExcludedNodes adds dependencies of the excluded vertex to the
// excludedNodes set. This includes all descendants in the graph.
func (t *ExcludesTransformer) addVertexDependenciesToExcludedNodes(g *Graph, v dag.Vertex, excludedNodes dag.VertexSet, addrs []addrs.Targetable) {
	if excludedNodes.Contains(v) {
		return
	}
	excludedNodes.Add(v)

	// TODO:@austinvalle: Consider nodes that could appear as descendants that we don't want to
	// exclude, for example: nodeCloseModule / policy?
	for d := range g.Descendants(v).All() {
		t.addVertexDependenciesToExcludedNodes(g, d, excludedNodes, addrs)
	}
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
)

// TargetsTransformer is a GraphTransformer that, when the user specifies a
// list of resources to target, limits the graph to only those resources and
// their dependencies.
type TargetsTransformer struct {
	// List of targeted resource names specified by the user
	Targets []addrs.Targetable
}

func (t *TargetsTransformer) Transform(g *Graph) error {
	if len(t.Targets) > 0 {
		_, targetedNodes := selectTargetedNodes(g, t.Targets)

		for _, v := range g.Vertices() {
			if !targetedNodes.Include(v) {
				log.Printf("[DEBUG] Removing %q, filtered by targeting.", dag.VertexName(v))
				g.Remove(v)
			}
		}
	}

	return nil
}

// Returns a set of targeted nodes. A targeted node is either addressed
// directly, address indirectly via its container, or it's a dependency of a
// targeted node.
func selectTargetedNodes(g *Graph, targets []addrs.Targetable) (dag.Set, dag.Set) {
	directNodes := make(dag.Set)
	targetedNodes := make(dag.Set)

	vertices := g.Vertices()
	set := addrs.MakeSet(targets...)
	for _, v := range vertices {
		if nodeIsTarget(v, set) {
			targetedNodes.Add(v)
			directNodes.Add(v)

			for _, d := range g.Ancestors(v) {
				targetedNodes.Add(d)
			}
		}
	}

	// It is expected that outputs which are only derived from targeted
	// resources are also updated. While we don't include any other possible
	// side effects from the targeted nodes, these are added because outputs
	// cannot be targeted on their own.
	// Start by finding the root module output nodes themselves
	for _, v := range vertices {
		// outputs are all temporary value types
		tv, ok := v.(graphNodeTemporaryValue)
		if !ok {
			continue
		}

		// root module outputs indicate that while they are an output type,
		// they not temporary and will return false here.
		if tv.temporaryValue() {
			continue
		}

		// If this output is descended only from targeted resources, then we
		// will keep it
		deps := g.Ancestors(v)
		found := 0
		for _, d := range deps {
			switch d.(type) {
			case GraphNodeResourceInstance:
			case GraphNodeConfigResource:
			default:
				continue
			}

			if !targetedNodes.Include(d) {
				// this dependency isn't being targeted, so we can't process this
				// output
				found = 0
				break
			}

			found++
		}

		if found > 0 {
			// we found an output we can keep; add it, and all it's dependencies
			targetedNodes.Add(v)
			for _, d := range deps {
				targetedNodes.Add(d)
			}
		}
	}

	return directNodes, targetedNodes
}

func nodeIsTarget(v dag.Vertex, targets addrs.Set[addrs.Targetable]) bool {
	var vertexAddr addrs.Targetable
	switch r := v.(type) {
	case *nodeApplyableDeferredPartialInstance:

		// Partial instances are not targeted directly, but they might be
		// targeted after they have been expanded so we need to perform a custom
		// check for them here.
		//
		// The other types of nodes can be targeted directly, and are handled
		// together.

		for _, targetAddr := range targets {
			if r.PartialAddr.IsTargetedBy(targetAddr) {
				return true
			}
		}
		return false

	case GraphNodeResourceInstance:
		vertexAddr = r.ResourceInstanceAddr()
	case GraphNodeConfigResource:
		vertexAddr = r.ResourceAddr()

	default:
		// Only partial nodes and resource and resource instance nodes can be
		// targeted.
		return false
	}

	for _, targetAddr := range targets {
		switch vertexAddr.(type) {
		case addrs.ConfigResource:
			// Before expansion happens, we only have nodes that know their
			// ConfigResource address.  We need to take the more specific
			// target addresses and generalize them in order to compare with a
			// ConfigResource.
			switch target := targetAddr.(type) {
			case addrs.AbsResourceInstance:
				targetAddr = target.ContainingResource().Config()
			case addrs.AbsResource:
				targetAddr = target.Config()
			case addrs.ModuleInstance:
				targetAddr = target.Module()
			}
		}

		if targetAddr.TargetContains(vertexAddr) {
			return true
		}
	}

	return false
}

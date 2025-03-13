// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
)

// getTargetable extracts the targetable address from a node. The order
// of the checks is important, as the GraphNodeResourceInstance takes precedence
// over the GraphNodeConfigResource.
func getTargetable(node dag.Vertex) addrs.Targetable {
	switch n := node.(type) {
	case GraphNodeResourceInstance:
		return n.ResourceInstanceAddr()
	case GraphNodeConfigResource:
		return n.ResourceAddr()
	case GraphNodeModulePath:
		return n.ModulePath()
	default:
		return nil
	}
}

// setContains checks if a given node or any of its ancestors are present
// in the set. It first checks if the node itself is excluded,
// and if not, it recursively checks all ancestor nodes.
func (g *Graph) setContains(node dag.Vertex, targets addrs.Set[addrs.Targetable]) bool {
	targetable := getTargetable(node)
	if targetable == nil {
		return false
	}

	contains := func(t addrs.Targetable) bool {
		for _, target := range targets {
			if target.TargetContains(t) {
				return true
			}
		}
		return false
	}

	if contains(targetable) {
		return true
	}

	for _, dep := range g.Ancestors(node) {
		if targetable := getTargetable(dep); targetable != nil && contains(targetable) {
			return true
		}
	}
	return false
}

// deferTargets processes the exclusion rules for the graph.
// It excludes any nodes that match the exclusion addresses or have excluded ancestors.
func (g *Graph) deferTargets(ctx EvalContext, deferredAddrs addrs.Set[addrs.Targetable]) dag.Set {
	// Note: If the node is a dynamic node, but the exclusion is for a more specific target,
	// the dynamic node will not be excluded, and that target will be excluded during
	// the dynamic expansion subgraph walk.
	for _, node := range g.Vertices() {
		// Skip nodes that are not deferrable
		node, ok := node.(GraphNodeDeferrable)
		if !ok {
			continue
		}

		// Check if this node should be deferred based on itself or its ancestors
		if g.setContains(node, deferredAddrs) {
			node.SetDeferred(true)
			continue
		}

		// Check if this node should be deferred based on its dependencies
		if gd, ok := node.(GD); ok {
			if ctx.Deferrals().DependenciesDeferred(gd.GetDependencies()) {
				node.SetDeferred(true)
				continue
			}
		}
	}
	return nil
}

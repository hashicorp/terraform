// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
)

// GraphNodeTargetable is an interface for graph nodes to implement when they
// need to be told about incoming targets. This is useful for nodes that need
// to respect targets as they dynamically expand. Note that the list of targets
// provided will contain every target provided, and each implementing graph
// node must filter this list to targets considered relevant.
type GraphNodeTargetable interface {
	SetTargets([]addrs.Targetable)
	SetExcludes([]addrs.Targetable)
}

// TargetsTransformer is a GraphTransformer that:
//   - when the user specifies a list of resources to target,
//     limits the graph to only those resources and their dependencies.
//   - when the user specifies a list of resources to exclude,
//     limits the graph to everything except those resources and their dependencies.
//
// Targets, Excludes, and ActionTargets are mutually exclusive
type TargetsTransformer struct {
	// List of targeted resource names specified by the user.
	Targets []addrs.Targetable

	// List of excluded resource names specified by the user.
	Excludes []addrs.Targetable

	// List of targeted actions specified by the user.
	ActionTargets []addrs.Targetable
}

func (t *TargetsTransformer) Transform(g *Graph) error {
	if len(t.Targets) == 0 && len(t.ActionTargets) == 0 && len(t.Excludes) == 0 {
		return nil
	}

	// in practice, these are mutually exclusive so only one of these function
	// calls will do any work

	targetedNodes := t.selectTargetedNodes(g, t.Targets)
	targetedActions := t.selectTargetedNodes(g, t.ActionTargets)
	excludedNodes := t.selectExcludedNodes(g, t.Excludes)
	for _, v := range g.Vertices() {
		// TODO: Exclude logic should probably just move to a different transformer since it barely has anything to do with
		// the rest of this file :P
		if len(t.Excludes) > 0 {
			if excludedNodes.Include(v) {
				log.Printf("[DEBUG] Removing %q, filtered by targeting (excluded).", dag.VertexName(v))
				g.Remove(v)
			}
			continue
		}

		if !targetedNodes.Include(v) && !targetedActions.Include(v) {
			log.Printf("[DEBUG] Removing %q, filtered by targeting.", dag.VertexName(v))
			g.Remove(v)
		}
	}

	return nil
}

// Returns a set of targeted nodes. A targeted node is either addressed
// directly, address indirectly via its container, or it's a dependency of a
// targeted node.
func (t *TargetsTransformer) selectTargetedNodes(g *Graph, addrs []addrs.Targetable) dag.Set {
	targetedNodes := make(dag.Set)
	if len(addrs) == 0 {
		return targetedNodes
	}

	vertices := g.Vertices()

	for _, v := range vertices {
		if t.nodeIsTarget(v, addrs) {
			// We need to add everything this node depends on or that is closely associated with
			// this node. In case of resource nodes, action triggers are considered closely related
			// since they belong to the resource.
			t.addVertexDependenciesToTargetedNodes(g, v, targetedNodes, addrs)

			// We inform nodes that ask about the list of targets - helps for nodes
			// that need to dynamically expand. Note that this only occurs for nodes
			// that are already directly targeted.
			if tn, ok := v.(GraphNodeTargetable); ok {
				tn.SetTargets(addrs)
			}

			if _, ok := v.(*nodeExpandPlannableResource); ok {
				// We want to also set the resource instance triggers on the related action triggers
				for _, d := range g.UpEdges(v) {
					if actionTrigger, ok := d.(*nodeActionTriggerPlanExpand); ok {
						actionTrigger.SetResourceTargets(addrs)
					}
				}
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

	return targetedNodes
}

func (t *TargetsTransformer) nodeIsTarget(v dag.Vertex, targets []addrs.Targetable) bool {
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
	case *nodeActionInvokeExpand:
		vertexAddr = r.Target
	case *nodeActionTriggerApplyInstance:
		vertexAddr = r.ActionInvocation.Addr

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

// addVertexDependenciesToTargetedNodes adds dependencies of the targeted vertex to the
// targetedNodes set. This includes all ancestors in the graph.
// It also includes all action trigger nodes in the graph. Actions are planned after the
// triggering node has planned so that we can ensure the actions are only planned if the triggering
// resource has an action (Create / Update) corresponding to one of the events in the action trigger
// blocks event list.
func (t *TargetsTransformer) addVertexDependenciesToTargetedNodes(g *Graph, v dag.Vertex, targetedNodes dag.Set, addrs []addrs.Targetable) {
	if targetedNodes.Include(v) {
		return
	}
	targetedNodes.Add(v)

	for _, d := range g.Ancestors(v) {
		t.addVertexDependenciesToTargetedNodes(g, d, targetedNodes, addrs)
	}

	if _, ok := v.(*nodeExpandPlannableResource); ok {
		// We want to also add the action triggers related to this resource
		for _, d := range g.UpEdges(v) {
			if _, ok := d.(*nodeActionTriggerPlanExpand); ok {
				t.addVertexDependenciesToTargetedNodes(g, d, targetedNodes, addrs)
			}
		}
	}

	// An applyable resources might have an associated after_* triggered action.
	// We need to add that action to the targeted nodes as well, together with all its dependencies.
	if _, ok := v.(*nodeExpandApplyableResource); ok {
		for _, f := range g.UpEdges(v) {
			if _, ok := f.(*nodeActionTriggerApplyExpand); ok {
				t.addVertexDependenciesToTargetedNodes(g, f, targetedNodes, addrs)
			}
		}
	}
	if _, ok := v.(*NodeApplyableResourceInstance); ok {
		for _, f := range g.UpEdges(v) {
			if _, ok := f.(*nodeActionTriggerApplyExpand); ok {
				t.addVertexDependenciesToTargetedNodes(g, f, targetedNodes, addrs)
			}
		}
	}
}

func (t *TargetsTransformer) selectExcludedNodes(g *Graph, addrs []addrs.Targetable) dag.Set {
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

			// TODO: What about actions? I think we'll want to also exclude action triggers but it's actively
			// getting refactored so I'm not really sure if/where they will be in the graph after that :P
		}
	}

	// TODO: What about outputs? Targeting has specialized logic for them, but I'm not sure we need that here since I believe they
	// should be excluded by just being a descendant.

	return excludedNodes
}

func (t *TargetsTransformer) nodeIsExcluded(v dag.Vertex, excludes []addrs.Targetable) bool {
	var vertexAddr addrs.Targetable
	switch r := v.(type) {
	case *nodeApplyableDeferredPartialInstance:
		// TODO: This is handled in targeting, although I'm not sure yet how/if we need to implement this for excluding
		//
		// for _, excludeAddr := range excludes {
		// 	if r.PartialAddr.IsTargetedBy(excludeAddr) {
		// 		return true
		// 	}
		// }

		return false

	case GraphNodeResourceInstance:
		vertexAddr = r.ResourceInstanceAddr()
	case GraphNodeConfigResource:
		vertexAddr = r.ResourceAddr()

	// TODO: What about actions? I think we'll want to also exclude action triggers but it's actively
	// getting refactored so I'm not really sure if/where they will be in the graph after that :P
	//
	// case *nodeActionInvokeExpand:
	// 	vertexAddr = r.Target
	// case *nodeActionTriggerApplyInstance:
	// 	vertexAddr = r.ActionInvocation.Addr

	default:
		// Only partial nodes and resource and resource instance nodes can be
		// targeted.
		return false
	}

	for _, excludeAddr := range excludes {
		// In the case of an absolute instance, we cannot exclude the node (or it's dependants) until expansion has occurred,
		// so we cannot generalize the excludeAddr like targeting does.
		if excludeAddr.TargetContains(vertexAddr) {
			return true
		}
	}

	return false
}

// addVertexDependenciesToExcludedNodes adds dependencies of the excluded vertex to the
// excludedNodes set. This includes all descendants in the graph.
func (t *TargetsTransformer) addVertexDependenciesToExcludedNodes(g *Graph, v dag.Vertex, excludedNodes dag.Set, addrs []addrs.Targetable) {
	if excludedNodes.Include(v) {
		return
	}
	excludedNodes.Add(v)

	for _, d := range g.Descendants(v) {
		t.addVertexDependenciesToExcludedNodes(g, d, excludedNodes, addrs)
	}

	// TODO: What about actions? I think we'll want to also exclude action triggers but it's actively
	// getting refactored so I'm not really sure if/where they will be in the graph after that :P
}

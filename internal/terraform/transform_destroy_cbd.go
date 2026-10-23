// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"
	"slices"

	"github.com/hashicorp/terraform/internal/dag"
)

// GraphNodeCreateBeforeDestroy must be implemented by nodes that might be
// create-before-destroy destroyers, or might plan a create-before-destroy
// action.
type GraphNodeCreateBeforeDestroy interface {
	// CreateBeforeDestroy returns true if this node represents a node
	// that is doing a CBD.
	CreateBeforeDestroy() bool

	// ForceCreateBeforeDestroy is called to change the CBD state of a node when
	// it wasn't set directly via configuration.
	ForceCreateBeforeDestroy()
}

// ForcedCBDTransformer detects when a particular CBD-able graph node has
// dependencies with another that has create_before_destroy set that require
// it to be forced on, and forces it on.
//
// This must be used in the plan graph builder to ensure that
// create_before_destroy settings are properly propagated before constructing
// the planned changes. This requires that the plannable resource nodes
// implement GraphNodeDestroyerCBD.
//
// This is not used in the apply graph builder, where the planned changes have
// already decided the create_before_destroy ordering, and the CBDEdgeTransformer
// handles the apply nodes.
type ForcedCBDTransformer struct {
}

func (t *ForcedCBDTransformer) Transform(g *Graph) error {
	for v := range g.VerticesSeq() {
		// first look for any node reporting itself as CreateBeforeDestroy
		dn, ok := v.(GraphNodeCreateBeforeDestroy)
		if !ok || !dn.CreateBeforeDestroy() {
			continue
		}

		// We now have a node v which is CreateBeforeDestroy, so we have to make
		// sure all its ancestors have the same ordering.
		for anc := range g.Ancestors(v).All() {
			ancCBD, ok := anc.(GraphNodeCreateBeforeDestroy)
			if !ok {
				continue
			}

			if !ancCBD.CreateBeforeDestroy() {
				log.Printf("[TRACE] ForcedCBDTransformer: forcing create_before_destroy for %q(%T) due to %q(%T)", anc.Name(), anc, v.Name(), v)
				ancCBD.ForceCreateBeforeDestroy()
			}
		}
	}

	return nil
}

// CBDEdgeTransformer modifies the edges of create-before-destroy ("CBD") nodes
// that went through the DestroyEdgeTransformer so that they will have the
// correct dependencies. There are a few parts to this:
//
//  1. With CBD, the destroy edge is inverted: the destroy depends on
//     the creation.
//
//  2. Destroy for A must depend on resources that depend on A. This is to
//     allow the destroy to only happen once nodes that depend on A successfully
//     update to A. Example: adding a web server updates the load balancer
//     before deleting the old web server.
//
//  3. Every destroy node which must follow a CBD destroy node is also forced
//     to be CBD, but a replacement is always ordered as decided by the plan,
//     which we detect via GraphNodeDeposedResourceInstanceObject. Edges from other
//     creators are only inverted if the creator is not already ordered after
//     the destroy node through other edges.
//
// Before ordering the destroy nodes, create_before_destroy is also forced on
// for every node depended on by a create_before_destroy node, so that the
// status is stored with the new state of each instance.
type CBDEdgeTransformer struct {
}

func (t *CBDEdgeTransformer) Transform(g *Graph) error {
	forceCBDUpdateCreateDependencies(g)

	// Find all the destroy nodes which must be create_before_destroy. This is
	// every destroy node reporting itself as create_before_destroy, along with
	// all the destroy nodes which must follow those.
	cbdDestroyNodes := dag.NewVertexSet()
	var initialCBDDestroyNodes []dag.Vertex
	for v := range g.VerticesSeq() {
		dn, ok := v.(interface {
			GraphNodeDestroyer
			GraphNodeCreateBeforeDestroy
		})

		if ok && dn.CreateBeforeDestroy() {
			cbdDestroyNodes.Add(v)
			initialCBDDestroyNodes = append(initialCBDDestroyNodes, v)
		}
	}

	for desc := range g.Descendants(initialCBDDestroyNodes...).All() {
		// Only destroy nodes can be forced at this point, because forcing a
		// creator would change how its planned replacement is applied. This
		// covers the case where an orphan needs to be considered CBD due to
		// surrounding changes in configuration
		if _, ok := desc.(GraphNodeDestroyer); !ok {
			continue
		}

		if cbdDesc, ok := desc.(GraphNodeCreateBeforeDestroy); ok {
			cbdDesc.ForceCreateBeforeDestroy()
			if !cbdDesc.CreateBeforeDestroy() {
				// this one couldn't be overridden, so we make sure to not add
				// it to the cbd set
				continue
			}
		}

		cbdDestroyNodes.Add(desc)
	}

	// Take out the edges from creators to those destroy nodes. The edge from a
	// node's own creator follows the planned replacement, and the others,
	// stored in removedEdges, are decided once only the edges which will not change
	// remain.
	var removedEdges []dag.Edge
	for v := range cbdDestroyNodes.All() {
		for src := range g.EdgesTo(v).All() {
			c, ok := src.(GraphNodeCreator)
			if !ok {
				continue
			}

			// First reverse the edges that are directly from creators to their
			// deposed instance nodes
			if c.CreateAddr().Equal(v.(GraphNodeDestroyer).DestroyAddr()) {
				// the diff transformer ensures that leftover state deposed
				// nodes are no longer CreateBeforeDestroy, so we know that all
				// deposed objects here will be reversed.
				if _, ok := v.(GraphNodeDeposedResourceInstanceObject); ok {
					log.Printf("[TRACE] CBDEdgeTransformer: reversing edge %s -> %s", src.Name(), v.Name())
					g.RemoveEdge(src, v)
					g.Connect(v, src)
				}
				continue
			}

			// now remove and record the other edges which aren't directly part
			// of a replacement, and we'll replace them below while checking
			// which of the remaining set should be reversed.
			g.RemoveEdge(src, v)
			removedEdges = append(removedEdges, dag.Edge{Source: src, Target: v})
		}
	}

	// collect everything waiting for a CBD destroy node. This will be all
	// destroy descendants, and related update or create nodes.
	waitingOnCBD := g.Descendants(slices.Collect(cbdDestroyNodes.All())...)

	// Indexes of the remaining removed edges which may need to be reversed due
	// to CBD dependencies. We start by assuming all edges will be reversed, but
	// skip the ones which contradict the current graph ordering.
	edgesToReverse := slices.Repeat([]bool{true}, len(removedEdges))

	// ancestors of creators walks the same direction as descendants of
	// destroyers.
	ancestorsByCreator := map[dag.Vertex]dag.VertexSet{}

	for i, e := range removedEdges {
		// If this creator does not already depend on any CBD destroy node,
		// it cannot depend on this edge's target. Keep the default reversal
		// without computing its ancestors.
		if !waitingOnCBD.Contains(e.Source) {
			continue
		}

		// We then check if the target is an ancestor of the source, in which
		// case reversing the edge would cause a cycle. This should only be
		// possible when the stored dependencies and ordering of state
		// contradict changes in the configuration. Normally we would want
		// cycles to surface in order to detect contradicting situations, but
		// since there's no way to directly assign ordering to instances which
		// are no longer configured, we accept this as a best effort to create a
		// deterministic replacement graph.
		ancestors, ok := ancestorsByCreator[e.Source]
		if !ok {
			ancestors = g.Ancestors(e.Source)
			ancestorsByCreator[e.Source] = ancestors
		}
		edgesToReverse[i] = !ancestors.Contains(e.Target)
	}

	// We decided all directions before restoring any pending edges, so that
	// reconnecting one cannot affect the decision for another.
	for i, e := range removedEdges {
		if edgesToReverse[i] {
			log.Printf("[TRACE] CBDEdgeTransformer: reversing edge %s -> %s", e.Source.Name(), e.Target.Name())
			g.Connect(e.Target, e.Source)
		} else {
			log.Printf("[WARN] CBDEdgeTransformer: not reversing edge %s -> %s, which is already ordered by other edges", e.Source.Name(), e.Target.Name())
			g.Connect(e.Source, e.Target)
		}
	}

	return nil
}

// forceCBDUpdateCreateDependencies forces create_before_destroy on for every
// node which is depended on by a create_before_destroy node. The plan has
// already made this decision (this function is analogous to the
// ForceCBDTransformer in the plan graph), which is reflected in the planned
// actions, but newly created instances have no prior state in which to store
// it, so it must be recalculated for the new state.
//
// Only the dependencies between non-destroy nodes are followed. The edges into
// destroy nodes come from the DestroyEdgeTransformer, and propagating
// create_before_destroy through those would force it in the wrong direction.
//
// FIXME: The forced status should not need to be recalculated during apply.
// Currently however, the instance object which stores the planned information
// is lost for newly created instances because it contains no state value.
func forceCBDUpdateCreateDependencies(g *Graph) {
	var stack []dag.Vertex
	for v := range g.VerticesSeq() {
		if dn, ok := v.(GraphNodeCreateBeforeDestroy); ok && dn.CreateBeforeDestroy() {
			stack = append(stack, v)
		}
	}

	seen := dag.NewVertexSet()
	for len(stack) > 0 {
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		for dep := range g.EdgesFrom(v).All() {
			if _, ok := dep.(GraphNodeDestroyer); ok || seen.Contains(dep) {
				continue
			}
			seen.Add(dep)
			stack = append(stack, dep)

			if dn, ok := dep.(GraphNodeCreateBeforeDestroy); ok && !dn.CreateBeforeDestroy() {
				log.Printf("[TRACE] CBDEdgeTransformer: forcing create_before_destroy for %q(%T) due to %q(%T)", dep.Name(), dep, v.Name(), v)
				dn.ForceCreateBeforeDestroy()
			}
		}
	}
}

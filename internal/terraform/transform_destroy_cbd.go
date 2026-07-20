// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/states"
)

// GraphNodeCreateBeforeDestroy must be implemented by nodes that might be
// create-before-destroy destroyers, or might plan a create-before-destroy
// action.
type GraphNodeCreateBeforeDestroy interface {
	// CreateBeforeDestroy returns true if this node represents a node
	// that is doing a CBD.
	CreateBeforeDestroy() bool

	// ForceCreateBeforeDestroy is called when the CBD state of a node
	// is changed dynamically. This can return an error if this isn't
	// allowed.
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
type ForcedCBDTransformer struct {
}

func (t *ForcedCBDTransformer) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		dn, ok := v.(GraphNodeCreateBeforeDestroy)
		if !ok {
			continue
		}

		if !dn.CreateBeforeDestroy() {
			// If there are no CBD decendant (dependent nodes), then we
			// do nothing here.
			if !t.hasCBDDescendant(g, v) {
				log.Printf("[TRACE] ForcedCBDTransformer: %q (%T) has no CBD descendant, so skipping", dag.VertexName(v), v)
				continue
			}

			// If this isn't naturally a CBD node, this means that an descendant is
			// and we need to auto-upgrade this node to CBD. We do this because
			// a CBD node depending on non-CBD will result in cycles. To avoid this,
			// we always attempt to upgrade it.
			log.Printf("[TRACE] ForcedCBDTransformer: forcing create_before_destroy for %q (%T)", dag.VertexName(v), v)
			dn.ForceCreateBeforeDestroy()
		} else {
			log.Printf("[TRACE] ForcedCBDTransformer: %q (%T) already has create_before_destroy set", dag.VertexName(v), v)
		}
	}
	return nil
}

// hasCBDDescendant returns true if any descendant (node that depends on this)
// has CBD set.
func (t *ForcedCBDTransformer) hasCBDDescendant(g *Graph, v dag.Vertex) bool {
	// in the normal case, we have a descendent config node somewhere that reports CBD
	if g.MatchDescendant(v, func(ov dag.Vertex) bool {
		dn, ok := ov.(GraphNodeCreateBeforeDestroy)
		if ok && dn.CreateBeforeDestroy() {
			// some descendant is CreateBeforeDestroy, so we need to follow suit
			log.Printf("[TRACE] ForcedCBDTransformer: %q has CBD descendant %q", dag.VertexName(v), dag.VertexName(ov))
			return true
		}
		return false
	}) {
		return true
	}

	// It's also possible that there are some orphaned CBD nodes from dependents
	// that no longer exist. These will be directly connected destroy nodes.
	for _, dep := range g.DownEdges(v) {
		if _, destroyer := dep.(GraphNodeDestroyer); !destroyer {
			continue
		}

		dn, ok := dep.(GraphNodeCreateBeforeDestroy)
		if ok && dn.CreateBeforeDestroy() {
			log.Printf("[TRACE] ForcedCBDTransformer: %q depends on CBD destroy node %q", dag.VertexName(v), dag.VertexName(dep))
			return true
		}
	}
	return false
}

// CBDEdgeTransformer modifies the edges of create-before-destroy ("CBD") nodes
// that went through the DestroyEdgeTransformer so that they will have the
// correct dependencies. There are two parts to this:
//
//  1. With CBD, the destroy edge is inverted: the destroy depends on
//     the creation.
//
//  2. Destroy for A must depend on resources that depend on A. This is to
//     allow the destroy to only happen once nodes that depend on A successfully
//     update to A. Example: adding a web server updates the load balancer
//     before deleting the old web server.
//
// This transformer requires that a previous transformer has already forced
// create_before_destroy on for nodes that are depended on by explicit CBD
// nodes. This is the logic in ForcedCBDTransformer, though in practice we
// will get here by recording the CBD-ness of each change in the plan during
// the plan walk and then forcing the nodes into the appropriate setting during
// DiffTransformer when building the apply graph.
type CBDEdgeTransformer struct {
	// Module and State are only needed to look up dependencies in
	// any way possible. Either can be nil if not available.
	Config *configs.Config
	State  *states.State

	// FIXME: This should optimally be decided entirely during plan, and then we
	// can rely on the planned changes to determine the CreateBeforeDestroy
	// status. This would require very careful auditing however, since not all
	// nodes are represented exactly in the changes, and the way
	// CreateBeforeDestroy propagates through the graph is extremely important
	// for correctness and to prevent cycles.
}

func (t *CBDEdgeTransformer) Transform(g *Graph) error {
	// Go through and reverse any destroy edges
	for _, v := range g.Vertices() {
		dn, ok := v.(GraphNodeCreateBeforeDestroy)
		if !ok {
			continue
		}
		if _, ok = v.(GraphNodeDestroyer); !ok {
			continue
		}

		if !dn.CreateBeforeDestroy() {
			continue
		}

		// Find the resource edges
		for _, e := range g.EdgesTo(v) {
			src := e.Source()

			// If source is a create node, invert the edge.
			// This covers both the node's own creator, as well as reversing
			// any dependants' edges.
			if _, ok := src.(GraphNodeCreator); ok {
				log.Printf("[TRACE] CBDEdgeTransformer: reversing edge %s -> %s", dag.VertexName(src), dag.VertexName(v))
				g.RemoveEdge(e)
				g.Connect(dag.BasicEdge(v, src))
			}
		}
	}
	return nil
}

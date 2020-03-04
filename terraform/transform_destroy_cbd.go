package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/states"
)

// GraphNodeDestroyerCBD must be implemented by nodes that might be
// create-before-destroy destroyers, or might plan a create-before-destroy
// action.
type GraphNodeDestroyerCBD interface {
	// CreateBeforeDestroy returns true if this node represents a node
	// that is doing a CBD.
	CreateBeforeDestroy() bool

	// ModifyCreateBeforeDestroy is called when the CBD state of a node
	// is changed dynamically. This can return an error if this isn't
	// allowed.
	ModifyCreateBeforeDestroy(bool) error
}

// GraphNodeAttachDestroyer is implemented by applyable nodes that have a
// companion destroy node. This allows the creation node to look up the status
// of the destroy node and determine if it needs to depose the existing state,
// or replace it.
// If a node is not marked as create-before-destroy in the configuration, but a
// dependency forces that status, only the destroy node will be aware of that
// status.
type GraphNodeAttachDestroyer interface {
	// AttachDestroyNode takes a destroy node and saves a reference to that
	// node in the receiver, so it can later check the status of
	// CreateBeforeDestroy().
	AttachDestroyNode(n GraphNodeDestroyerCBD)
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
		dn, ok := v.(GraphNodeDestroyerCBD)
		if !ok {
			continue
		}

		if !dn.CreateBeforeDestroy() {
			// If there are no CBD decendent (dependent nodes), then we
			// do nothing here.
			if !t.hasCBDDescendent(g, v) {
				log.Printf("[TRACE] ForcedCBDTransformer: %q (%T) has no CBD descendent, so skipping", dag.VertexName(v), v)
				continue
			}

			// If this isn't naturally a CBD node, this means that an descendent is
			// and we need to auto-upgrade this node to CBD. We do this because
			// a CBD node depending on non-CBD will result in cycles. To avoid this,
			// we always attempt to upgrade it.
			log.Printf("[TRACE] ForcedCBDTransformer: forcing create_before_destroy on for %q (%T)", dag.VertexName(v), v)
			if err := dn.ModifyCreateBeforeDestroy(true); err != nil {
				return fmt.Errorf(
					"%s: must have create before destroy enabled because "+
						"a dependent resource has CBD enabled. However, when "+
						"attempting to automatically do this, an error occurred: %s",
					dag.VertexName(v), err)
			}
		} else {
			log.Printf("[TRACE] ForcedCBDTransformer: %q (%T) already has create_before_destroy set", dag.VertexName(v), v)
		}
	}
	return nil
}

// hasCBDDescendent returns true if any descendent (node that depends on this)
// has CBD set.
func (t *ForcedCBDTransformer) hasCBDDescendent(g *Graph, v dag.Vertex) bool {
	s, _ := g.Descendents(v)
	if s == nil {
		return true
	}

	for _, ov := range s {
		dn, ok := ov.(GraphNodeDestroyerCBD)
		if !ok {
			continue
		}

		if dn.CreateBeforeDestroy() {
			// some descendent is CreateBeforeDestroy, so we need to follow suit
			log.Printf("[TRACE] ForcedCBDTransformer: %q has CBD descendent %q", dag.VertexName(v), dag.VertexName(ov))
			return true
		}
	}

	return false
}

// CBDEdgeTransformer modifies the edges of CBD nodes that went through
// the DestroyEdgeTransformer to have the right dependencies. There are
// two real tasks here:
//
//   1. With CBD, the destroy edge is inverted: the destroy depends on
//      the creation.
//
//   2. A_d must depend on resources that depend on A. This is to enable
//      the destroy to only happen once nodes that depend on A successfully
//      update to A. Example: adding a web server updates the load balancer
//      before deleting the old web server.
//
// This transformer requires that a previous transformer has already forced
// create_before_destroy on for nodes that are depended on by explicit CBD
// nodes. This is the logic in ForcedCBDTransformer, though in practice we
// will get here by recording the CBD-ness of each change in the plan during
// the plan walk and then forcing the nodes into the appropriate setting during
// DiffTransformer when building the apply graph.
type CBDEdgeTransformer struct {
	// Module and State are only needed to look up dependencies in
	// any way possible. Either can be nil if not availabile.
	Config *configs.Config
	State  *states.State

	// If configuration is present then Schemas is required in order to
	// obtain schema information from providers and provisioners so we can
	// properly resolve implicit dependencies.
	Schemas *Schemas
}

func (t *CBDEdgeTransformer) Transform(g *Graph) error {
	// Go through and reverse any destroy edges
	for _, v := range g.Vertices() {
		dn, ok := v.(GraphNodeDestroyerCBD)
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

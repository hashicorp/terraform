package terraform

import (
	"log"

	"github.com/hashicorp/terraform/config/module"
)

// GraphNodeDestroyerCBD must be implemented by nodes that might be
// create-before-destroy destroyers.
type GraphNodeDestroyerCBD interface {
	GraphNodeDestroyer

	// CreateBeforeDestroy returns true if this node represents a node
	// that is doing a CBD.
	CreateBeforeDestroy() bool
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
type CBDEdgeTransformer struct {
	// Module and State are only needed to look up dependencies in
	// any way possible. Either can be nil if not availabile.
	Module *module.Tree
	State  *State
}

func (t *CBDEdgeTransformer) Transform(g *Graph) error {
	log.Printf("[TRACE] CBDEdgeTransformer: Beginning CBD transformation...")

	// Go through and reverse any destroy edges
	for _, v := range g.Vertices() {
		dn, ok := v.(GraphNodeDestroyerCBD)
		if !ok {
			continue
		}

		if !dn.CreateBeforeDestroy() {
			continue
		}

		// Find the destroy edge. There should only be one.
		for _, e := range g.EdgesTo(v) {
			log.Printf("WHAT: %#v", e)
			// Not a destroy edge, ignore it
			de, ok := e.(*DestroyEdge)
			if !ok {
				continue
			}

			// Found it! Invert.
			g.RemoveEdge(de)
			g.Connect(&DestroyEdge{S: de.Target(), T: de.Source()})
		}
	}

	return nil
}

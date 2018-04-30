package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeDestroyerCBD must be implemented by nodes that might be
// create-before-destroy destroyers.
type GraphNodeDestroyerCBD interface {
	GraphNodeDestroyer

	// CreateBeforeDestroy returns true if this node represents a node
	// that is doing a CBD.
	CreateBeforeDestroy() bool

	// ModifyCreateBeforeDestroy is called when the CBD state of a node
	// is changed dynamically. This can return an error if this isn't
	// allowed.
	ModifyCreateBeforeDestroy(bool) error
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
	Config *configs.Config
	State  *State
}

func (t *CBDEdgeTransformer) Transform(g *Graph) error {
	log.Printf("[TRACE] CBDEdgeTransformer: Beginning CBD transformation...")

	// Go through and reverse any destroy edges
	destroyMap := make(map[string][]dag.Vertex)
	for _, v := range g.Vertices() {
		dn, ok := v.(GraphNodeDestroyerCBD)
		if !ok {
			continue
		}

		if !dn.CreateBeforeDestroy() {
			// If there are no CBD ancestors (dependent nodes), then we
			// do nothing here.
			if !t.hasCBDAncestor(g, v) {
				continue
			}

			// If this isn't naturally a CBD node, this means that an ancestor is
			// and we need to auto-upgrade this node to CBD. We do this because
			// a CBD node depending on non-CBD will result in cycles. To avoid this,
			// we always attempt to upgrade it.
			if err := dn.ModifyCreateBeforeDestroy(true); err != nil {
				return fmt.Errorf(
					"%s: must have create before destroy enabled because "+
						"a dependent resource has CBD enabled. However, when "+
						"attempting to automatically do this, an error occurred: %s",
					dag.VertexName(v), err)
			}
		}

		// Find the destroy edge. There should only be one.
		for _, e := range g.EdgesTo(v) {
			// Not a destroy edge, ignore it
			de, ok := e.(*DestroyEdge)
			if !ok {
				continue
			}

			log.Printf("[TRACE] CBDEdgeTransformer: inverting edge: %s => %s",
				dag.VertexName(de.Source()), dag.VertexName(de.Target()))

			// Found it! Invert.
			g.RemoveEdge(de)
			g.Connect(&DestroyEdge{S: de.Target(), T: de.Source()})
		}

		// If the address has an index, we strip that. Our depMap creation
		// graph doesn't expand counts so we don't currently get _exact_
		// dependencies. One day when we limit dependencies more exactly
		// this will have to change. We have a test case covering this
		// (depNonCBDCountBoth) so it'll be caught.
		addr := dn.DestroyAddr()
		key := addr.ContainingResource().String()

		// Add this to the list of nodes that we need to fix up
		// the edges for (step 2 above in the docs).
		destroyMap[key] = append(destroyMap[key], v)
	}

	// If we have no CBD nodes, then our work here is done
	if len(destroyMap) == 0 {
		return nil
	}

	// We have CBD nodes. We now have to move on to the much more difficult
	// task of connecting dependencies of the creation side of the destroy
	// to the destruction node. The easiest way to explain this is an example:
	//
	// Given a pre-destroy dependence of: A => B
	//   And A has CBD set.
	//
	// The resulting graph should be: A => B => A_d
	//
	// They key here is that B happens before A is destroyed. This is to
	// facilitate the primary purpose for CBD: making sure that downstreams
	// are properly updated to avoid downtime before the resource is destroyed.
	//
	// We can't trust that the resource being destroyed or anything that
	// depends on it is actually in our current graph so we make a new
	// graph in order to determine those dependencies and add them in.
	log.Printf("[TRACE] CBDEdgeTransformer: building graph to find dependencies...")
	depMap, err := t.depMap(destroyMap)
	if err != nil {
		return err
	}

	// We now have the mapping of resource addresses to the destroy
	// nodes they need to depend on. We now go through our own vertices to
	// find any matching these addresses and make the connection.
	for _, v := range g.Vertices() {
		// We're looking for creators
		rn, ok := v.(GraphNodeCreator)
		if !ok {
			continue
		}

		// Get the address
		addr := rn.CreateAddr()

		// If the address has an index, we strip that. Our depMap creation
		// graph doesn't expand counts so we don't currently get _exact_
		// dependencies. One day when we limit dependencies more exactly
		// this will have to change. We have a test case covering this
		// (depNonCBDCount) so it'll be caught.
		key := addr.ContainingResource().String()

		// If there is nothing this resource should depend on, ignore it
		dns, ok := depMap[key]
		if !ok {
			continue
		}

		// We have nodes! Make the connection
		for _, dn := range dns {
			log.Printf("[TRACE] CBDEdgeTransformer: destroy depends on dependence: %s => %s",
				dag.VertexName(dn), dag.VertexName(v))
			g.Connect(dag.BasicEdge(dn, v))
		}
	}

	return nil
}

func (t *CBDEdgeTransformer) depMap(destroyMap map[string][]dag.Vertex) (map[string][]dag.Vertex, error) {
	// Build the graph of our config, this ensures that all resources
	// are present in the graph.
	g, diags := (&BasicGraphBuilder{
		Steps: []GraphTransformer{
			&FlatConfigTransformer{Config: t.Config},
			&AttachResourceConfigTransformer{Config: t.Config},
			&AttachStateTransformer{State: t.State},
			&ReferenceTransformer{},
		},
		Name: "CBDEdgeTransformer",
	}).Build(nil)
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	// Using this graph, build the list of destroy nodes that each resource
	// address should depend on. For example, when we find B, we map the
	// address of B to A_d in the "depMap" variable below.
	depMap := make(map[string][]dag.Vertex)
	for _, v := range g.Vertices() {
		// We're looking for resources.
		rn, ok := v.(GraphNodeResource)
		if !ok {
			continue
		}

		// Get the address
		addr := rn.ResourceAddr()
		key := addr.String()

		// Get the destroy nodes that are destroying this resource.
		// If there aren't any, then we don't need to worry about
		// any connections.
		dns, ok := destroyMap[key]
		if !ok {
			continue
		}

		// Get the nodes that depend on this on. In the example above:
		// finding B in A => B.
		for _, v := range g.UpEdges(v).List() {
			// We're looking for resources.
			rn, ok := v.(GraphNodeResource)
			if !ok {
				continue
			}

			// Keep track of the destroy nodes that this address
			// needs to depend on.
			key := rn.ResourceAddr().String()
			depMap[key] = append(depMap[key], dns...)
		}
	}

	return depMap, nil
}

// hasCBDAncestor returns true if any ancestor (node that depends on this)
// has CBD set.
func (t *CBDEdgeTransformer) hasCBDAncestor(g *Graph, v dag.Vertex) bool {
	s, _ := g.Ancestors(v)
	if s == nil {
		return true
	}

	for _, v := range s.List() {
		dn, ok := v.(GraphNodeDestroyerCBD)
		if !ok {
			continue
		}

		if dn.CreateBeforeDestroy() {
			// some ancestor is CreateBeforeDestroy, so we need to follow suit
			return true
		}
	}

	return false
}

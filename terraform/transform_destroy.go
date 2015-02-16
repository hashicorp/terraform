package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/dag"
)

// GraphNodeDestroyable is the interface that nodes that can be destroyed
// must implement. This is used to automatically handle the creation of
// destroy nodes in the graph and the dependency ordering of those destroys.
type GraphNodeDestroyable interface {
	// DestroyNode returns the node used for the destroy. This should
	// return the same node every time so that it can be used later for
	// lookups as well.
	DestroyNode() GraphNodeDestroy
}

// GraphNodeDestroy is the interface that must implemented by
// nodes that destroy.
type GraphNodeDestroy interface {
	dag.Vertex

	// CreateBeforeDestroy is called to check whether this node
	// should be created before it is destroyed. The CreateBeforeDestroy
	// transformer uses this information to setup the graph.
	CreateBeforeDestroy() bool

	// CreateNode returns the node used for the create side of this
	// destroy. This must already exist within the graph.
	CreateNode() dag.Vertex
}

// GraphNodeDiffPrunable is the interface that can be implemented to
// signal that this node can be pruned depending on what is in the diff.
type GraphNodeDiffPrunable interface {
	// DiffId is used to return the ID that should be checked for
	// pruning this resource. If this is empty, pruning won't be done.
	DiffId() string
}

// DestroyTransformer is a GraphTransformer that creates the destruction
// nodes for things that _might_ be destroyed.
type DestroyTransformer struct{}

func (t *DestroyTransformer) Transform(g *Graph) error {
	nodes := make(map[dag.Vertex]struct{}, len(g.Vertices()))
	for _, v := range g.Vertices() {
		// If it is not a destroyable, we don't care
		dn, ok := v.(GraphNodeDestroyable)
		if !ok {
			continue
		}

		// Grab the destroy side of the node and connect it through
		n := dn.DestroyNode()
		if n == nil {
			continue
		}

		// Store it
		nodes[n] = struct{}{}

		// Add it to the graph
		g.Add(n)

		// Inherit all the edges from the old node
		downEdges := g.DownEdges(v).List()
		for _, edgeRaw := range downEdges {
			g.Connect(dag.BasicEdge(n, edgeRaw.(dag.Vertex)))
		}

		// Add a new edge to connect the node to be created to
		// the destroy node.
		g.Connect(dag.BasicEdge(v, n))
	}

	// Go through the nodes we added and determine if they depend
	// on any nodes with a destroy node. If so, depend on that instead.
	for n, _ := range nodes {
		for _, downRaw := range g.DownEdges(n).List() {
			target := downRaw.(dag.Vertex)
			dn, ok := target.(GraphNodeDestroyable)
			if !ok {
				continue
			}

			newTarget := dn.DestroyNode()
			if newTarget == nil {
				continue
			}

			if _, ok := nodes[newTarget]; !ok {
				return fmt.Errorf(
					"%s: didn't generate same DestroyNode: %s",
					dag.VertexName(target),
					dag.VertexName(newTarget))
			}

			// Make the new edge and transpose
			g.Connect(dag.BasicEdge(newTarget, n))

			// Remove the old edge
			g.RemoveEdge(dag.BasicEdge(n, target))
		}
	}

	return nil
}

// CreateBeforeDestroyTransformer is a GraphTransformer that modifies
// the destroys of some nodes so that the creation happens before the
// destroy.
type CreateBeforeDestroyTransformer struct{}

func (t *CreateBeforeDestroyTransformer) Transform(g *Graph) error {
	// We "stage" the edge connections/destroys in these slices so that
	// while we're doing the edge transformations (transpositions) in
	// the graph, we're not affecting future edge transpositions. These
	// slices let us stage ALL the changes that WILL happen so that all
	// of the transformations happen atomically.
	var connect, destroy []dag.Edge

	for _, v := range g.Vertices() {
		// We only care to use the destroy nodes
		dn, ok := v.(GraphNodeDestroy)
		if !ok {
			continue
		}

		// If the node doesn't need to create before destroy, then continue
		if !dn.CreateBeforeDestroy() {
			continue
		}

		// Get the creation side of this node
		cn := dn.CreateNode()

		// Take all the things which depend on the creation node and
		// make them dependencies on the destruction. Clarifying this
		// with an example: if you have a web server and a load balancer
		// and the load balancer depends on the web server, then when we
		// do a create before destroy, we want to make sure the steps are:
		//
		// 1.) Create new web server
		// 2.) Update load balancer
		// 3.) Delete old web server
		//
		// This ensures that.
		for _, sourceRaw := range g.UpEdges(cn).List() {
			source := sourceRaw.(dag.Vertex)
			connect = append(connect, dag.BasicEdge(dn, source))
		}

		// Swap the edge so that the destroy depends on the creation
		// happening...
		connect = append(connect, dag.BasicEdge(dn, cn))
		destroy = append(destroy, dag.BasicEdge(cn, dn))
	}

	for _, edge := range connect {
		g.Connect(edge)
	}
	for _, edge := range destroy {
		g.RemoveEdge(edge)
	}

	return nil
}

// PruneDestroyTransformer is a GraphTransformer that removes the destroy
// nodes that aren't in the diff.
type PruneDestroyTransformer struct {
	Diff *Diff
}

func (t *PruneDestroyTransformer) Transform(g *Graph) error {
	var modDiff *ModuleDiff
	if t.Diff != nil {
		modDiff = t.Diff.ModuleByPath(g.Path)
	}

	for _, v := range g.Vertices() {
		// If it is not a destroyer, we don't care
		dn, ok := v.(GraphNodeDiffPrunable)
		if !ok {
			continue
		}

		// Grab the name to destroy
		prefix := dn.DiffId()
		if prefix == "" {
			continue
		}

		remove := true
		if modDiff != nil {
			for k, _ := range modDiff.Resources {
				if strings.HasPrefix(k, prefix) {
					remove = false
					break
				}
			}
		}

		// Remove the node if we have to
		if remove {
			g.Remove(v)
		}
	}

	return nil
}

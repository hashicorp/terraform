package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

type GraphNodeDestroyMode byte

const (
	DestroyNone    GraphNodeDestroyMode = 0
	DestroyPrimary GraphNodeDestroyMode = 1 << iota
	DestroyTainted
)

// GraphNodeDestroyable is the interface that nodes that can be destroyed
// must implement. This is used to automatically handle the creation of
// destroy nodes in the graph and the dependency ordering of those destroys.
type GraphNodeDestroyable interface {
	// DestroyNode returns the node used for the destroy with the given
	// mode. If this returns nil, then a destroy node for that mode
	// will not be added.
	DestroyNode(GraphNodeDestroyMode) GraphNodeDestroy
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

// GraphNodeDestroyPrunable is the interface that can be implemented to
// signal that this node can be pruned depending on state.
type GraphNodeDestroyPrunable interface {
	// DestroyInclude is called to check if this node should be included
	// with the given state. The state and diff must NOT be modified.
	DestroyInclude(*ModuleDiff, *ModuleState) bool
}

// GraphNodeEdgeInclude can be implemented to not include something
// as an edge within the destroy graph. This is usually done because it
// might cause unnecessary cycles.
type GraphNodeDestroyEdgeInclude interface {
	DestroyEdgeInclude(dag.Vertex) bool
}

// DestroyTransformer is a GraphTransformer that creates the destruction
// nodes for things that _might_ be destroyed.
type DestroyTransformer struct {
	FullDestroy bool
}

func (t *DestroyTransformer) Transform(g *Graph) error {
	var connect, remove []dag.Edge

	modes := []GraphNodeDestroyMode{DestroyPrimary, DestroyTainted}
	for _, m := range modes {
		connectMode, removeMode, err := t.transform(g, m)
		if err != nil {
			return err
		}

		connect = append(connect, connectMode...)
		remove = append(remove, removeMode...)
	}

	// Atomatically add/remove the edges
	for _, e := range connect {
		g.Connect(e)
	}
	for _, e := range remove {
		g.RemoveEdge(e)
	}

	return nil
}

func (t *DestroyTransformer) transform(
	g *Graph, mode GraphNodeDestroyMode) ([]dag.Edge, []dag.Edge, error) {
	var connect, remove []dag.Edge
	nodeToCn := make(map[dag.Vertex]dag.Vertex, len(g.Vertices()))
	nodeToDn := make(map[dag.Vertex]dag.Vertex, len(g.Vertices()))
	for _, v := range g.Vertices() {
		// If it is not a destroyable, we don't care
		cn, ok := v.(GraphNodeDestroyable)
		if !ok {
			continue
		}

		// Grab the destroy side of the node and connect it through
		n := cn.DestroyNode(mode)
		if n == nil {
			continue
		}

		// Store it
		nodeToCn[n] = cn
		nodeToDn[cn] = n

		// If the creation node is equal to the destroy node, then
		// don't do any of the edge jump rope below.
		if n.(interface{}) == cn.(interface{}) {
			continue
		}

		// Add it to the graph
		g.Add(n)

		// Inherit all the edges from the old node
		downEdges := g.DownEdges(v).List()
		for _, edgeRaw := range downEdges {
			// If this thing specifically requests to not be depended on
			// by destroy nodes, then don't.
			if i, ok := edgeRaw.(GraphNodeDestroyEdgeInclude); ok &&
				!i.DestroyEdgeInclude(v) {
				continue
			}

			g.Connect(dag.BasicEdge(n, edgeRaw.(dag.Vertex)))
		}

		// Add a new edge to connect the node to be created to
		// the destroy node.
		connect = append(connect, dag.BasicEdge(v, n))
	}

	// Go through the nodes we added and determine if they depend
	// on any nodes with a destroy node. If so, depend on that instead.
	for n, _ := range nodeToCn {
		for _, downRaw := range g.DownEdges(n).List() {
			target := downRaw.(dag.Vertex)
			cn2, ok := target.(GraphNodeDestroyable)
			if !ok {
				continue
			}

			newTarget := nodeToDn[cn2]
			if newTarget == nil {
				continue
			}

			// Make the new edge and transpose
			connect = append(connect, dag.BasicEdge(newTarget, n))

			// Remove the old edge
			remove = append(remove, dag.BasicEdge(n, target))
		}
	}

	return connect, remove, nil
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

			// If the graph has a "root" node (one added by a RootTransformer and not
			// just a resource that happens to have no ancestors), we don't want to
			// add any edges to it, because then it ceases to be a root.
			if _, ok := source.(graphNodeRoot); ok {
				continue
			}

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
	Diff  *Diff
	State *State
}

func (t *PruneDestroyTransformer) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		// If it is not a destroyer, we don't care
		dn, ok := v.(GraphNodeDestroyPrunable)
		if !ok {
			continue
		}

		path := g.Path
		if pn, ok := v.(GraphNodeSubPath); ok {
			path = pn.Path()
		}

		var modDiff *ModuleDiff
		var modState *ModuleState
		if t.Diff != nil {
			modDiff = t.Diff.ModuleByPath(path)
		}
		if t.State != nil {
			modState = t.State.ModuleByPath(path)
		}

		// Remove it if we should
		if !dn.DestroyInclude(modDiff, modState) {
			g.Remove(v)
		}
	}

	return nil
}

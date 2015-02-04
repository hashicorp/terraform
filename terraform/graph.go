package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/dag"
)

// RootModuleName is the name given to the root module implicitly.
const RootModuleName = "root"

// RootModulePath is the path for the root module.
var RootModulePath = []string{RootModuleName}

// Graph represents the graph that Terraform uses to represent resources
// and their dependencies. Each graph represents only one module, but it
// can contain further modules, which themselves have their own graph.
type Graph struct {
	// Graph is the actual DAG. This is embedded so you can call the DAG
	// methods directly.
	dag.AcyclicGraph

	// Path is the path in the module tree that this Graph represents.
	// The root is represented by a single element list containing
	// RootModuleName
	Path []string

	// dependableMap is a lookaside table for fast lookups for connecting
	// dependencies by their GraphNodeDependable value to avoid O(n^3)-like
	// situations and turn them into O(1) with respect to the number of new
	// edges.
	dependableMap map[string]dag.Vertex

	once sync.Once
}

// Add is the same as dag.Graph.Add.
func (g *Graph) Add(v dag.Vertex) dag.Vertex {
	g.once.Do(g.init)

	// Call upwards to add it to the actual graph
	g.Graph.Add(v)

	// If this is a depend-able node, then store the lookaside info
	if dv, ok := v.(GraphNodeDependable); ok {
		for _, n := range dv.DependableName() {
			g.dependableMap[n] = v
		}
	}

	return v
}

// ConnectDependent connects a GraphNodeDependent to all of its
// GraphNodeDependables. It returns the list of dependents it was
// unable to connect to.
func (g *Graph) ConnectDependent(raw dag.Vertex) []string {
	v, ok := raw.(GraphNodeDependent)
	if !ok {
		return nil
	}

	return g.ConnectTo(v, v.DependentOn())
}

// ConnectDependents goes through the graph, connecting all the
// GraphNodeDependents to GraphNodeDependables. This is safe to call
// multiple times.
//
// To get details on whether dependencies could be found/made, the more
// specific ConnectDependent should be used.
func (g *Graph) ConnectDependents() {
	for _, v := range g.Vertices() {
		if dv, ok := v.(GraphNodeDependent); ok {
			g.ConnectDependent(dv)
		}
	}
}

// ConnectFrom creates an edge by finding the source from a DependableName
// and connecting it to the specific vertex.
func (g *Graph) ConnectFrom(source string, target dag.Vertex) {
	g.once.Do(g.init)

	if source := g.dependableMap[source]; source != nil {
		g.Connect(dag.BasicEdge(source, target))
	}
}

// ConnectTo connects a vertex to a raw string of targets that are the
// result of DependableName, and returns the list of targets that are missing.
func (g *Graph) ConnectTo(v dag.Vertex, targets []string) []string {
	g.once.Do(g.init)

	var missing []string
	for _, t := range targets {
		if dest := g.dependableMap[t]; dest != nil {
			g.Connect(dag.BasicEdge(v, dest))
		} else {
			missing = append(missing, t)
		}
	}

	return missing
}

// Dependable finds the vertices in the graph that have the given dependable
// names and returns them.
func (g *Graph) Dependable(n string) dag.Vertex {
	// TODO: do we need this?
	return nil
}

func (g *Graph) init() {
	if g.dependableMap == nil {
		g.dependableMap = make(map[string]dag.Vertex)
	}
}

// GraphNodeDependable is an interface which says that a node can be
// depended on (an edge can be placed between this node and another) according
// to the well-known name returned by DependableName.
//
// DependableName can return multiple names it is known by.
type GraphNodeDependable interface {
	DependableName() []string
}

// GraphNodeDependent is an interface which says that a node depends
// on another GraphNodeDependable by some name. By implementing this
// interface, Graph.ConnectDependents() can be called multiple times
// safely and efficiently.
type GraphNodeDependent interface {
	DependentOn() []string
}

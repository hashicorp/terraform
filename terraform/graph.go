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
	*dag.Graph

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

// ConnectTo is a helper to create edges between a node and a list of
// targets by their DependableNames.
func (g *Graph) ConnectTo(source dag.Vertex, target []string) []string {
	g.once.Do(g.init)

	var missing []string
	for _, t := range target {
		if dest := g.dependableMap[t]; dest != nil {
			g.Connect(dag.BasicEdge(source, dest))
		} else {
			missing = append(missing, t)
		}
	}

	return missing
}

func (g *Graph) init() {
	if g.Graph == nil {
		g.Graph = new(dag.Graph)
	}

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

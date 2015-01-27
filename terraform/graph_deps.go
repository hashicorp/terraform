package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeDependable is an interface which says that a node can be
// depended on (an edge can be placed between this node and another) according
// to the well-known name returned by DependableName.
//
// DependableName can return multiple names it is known by.
type GraphNodeDependable interface {
	DependableName() []string
}

// GraphConnectDeps is a helper to connect a Vertex to the proper dependencies
// in the graph based only on the names expected by DependableName.
//
// This function will return the number of dependencies found and connected.
func GraphConnectDeps(g *dag.Graph, source dag.Vertex, targets []string) int {
	count := 0

	// This is reasonably horrible. In the future, we should optimize this
	// through some kind of metadata on the graph that can store all of
	// this information in a look-aside table.
	for _, v := range g.Vertices() {
		if dv, ok := v.(GraphNodeDependable); ok {
			for _, n := range dv.DependableName() {
				for _, n2 := range targets {
					if n == n2 {
						count++
						g.Connect(dag.BasicEdge(source, v))
						goto NEXT
					}
				}
			}
		}

	NEXT:
	}

	return count
}

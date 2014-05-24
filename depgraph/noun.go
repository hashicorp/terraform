package depgraph

import (
	"github.com/hashicorp/terraform/digraph"
)

// Nouns are the key structure of the dependency graph. They can
// be used to represent all objects in the graph. They are linked
// by depedencies.
type Noun struct {
	Name string // Opaque name
	Meta interface{}
	Deps []*Dependency
}

// Edges returns the out-going edges of a Noun
func (n *Noun) Edges() []digraph.Edge {
	edges := make([]digraph.Edge, len(n.Deps))
	for idx, dep := range n.Deps {
		edges[idx] = dep
	}
	return edges
}

func (n *Noun) String() string {
	return n.Name
}

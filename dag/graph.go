package dag

import (
	"bytes"
	"fmt"
	"sort"
)

// Graph is used to represent a dependency graph.
type Graph struct {
	Nodes []Node
}

// Node is an element of the graph that has other dependencies.
type Node interface {
	Deps() []Node
}

// NamedNode is an optional interface implementation of a Node that
// can have a name. If this is implemented, this will be used for various
// output.
type NamedNode interface {
	Node
	Name() string
}

func (g *Graph) String() string {
	var buf bytes.Buffer

	// Build the list of node names and a mapping so that we can more
	// easily alphabetize the output to remain deterministic.
	names := make([]string, 0, len(g.Nodes))
	mapping := make(map[string]Node, len(g.Nodes))
	for _, n := range g.Nodes {
		name := nodeName(n)
		names = append(names, name)
		mapping[name] = n
	}
	sort.Strings(names)

	// Write each node in order...
	for _, name := range names {
		n := mapping[name]
		buf.WriteString(fmt.Sprintf("%s\n", name))

		// Alphabetize dependencies
		depsRaw := n.Deps()
		deps := make([]string, 0, len(depsRaw))
		for _, d := range depsRaw {
			deps = append(deps, nodeName(d))
		}
		sort.Strings(deps)

		// Write dependencies
		for _, d := range deps {
			buf.WriteString(fmt.Sprintf("  %s\n", d))
		}
	}

	return buf.String()
}

func nodeName(n Node) string {
	switch v := n.(type) {
	case NamedNode:
		return v.Name()
	default:
		return fmt.Sprintf("%s", v)
	}
}

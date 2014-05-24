package digraph

// Digraph is used to represent a Directed Graph. This means
// we have a set of nodes, and a set of edges which are directed
// from a source and towards a destination
type Digraph interface {
	// Nodes provides all the nodes in the graph
	Nodes() []Node

	// Sources provides all the source nodes in the graph
	Sources() []Node

	// Sinks provides all the sink nodes in the graph
	Sinks() []Node

	// Transpose reverses the edge directions and returns
	// a new Digraph
	Transpose() Digraph
}

// Node represents a vertex in a Digraph
type Node interface {
	// Edges returns the out edges for a given nod
	Edges() []Edge
}

// Edge represents a directed edge in a Digraph
type Edge interface {
	// Head returns the start point of the Edge
	Head() Node

	// Tail returns the end point of the Edge
	Tail() Node
}

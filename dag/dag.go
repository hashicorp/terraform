package dag

// AcyclicGraph is a specialization of Graph that cannot have cycles. With
// this property, we get the property of sane graph traversal.
type AcyclicGraph struct {
	*Graph
}

// WalkFunc is the callback used for walking the graph.
type WalkFunc func(Vertex)

// Walk walks the graph, calling your callback as each node is visited.
func (g *AcyclicGraph) Walk(cb WalkFunc) {
}

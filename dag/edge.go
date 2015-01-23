package dag

// Edge represents an edge in the graph, with a source and target vertex.
type Edge interface {
	Source() Vertex
	Target() Vertex
}

// BasicEdge returns an Edge implementation that simply tracks the source
// and target given as-is.
func BasicEdge(source, target Vertex) Edge {
	return &basicEdge{S: source, T: target}
}

// basicEdge is a basic implementation of Edge that has the source and
// target vertex.
type basicEdge struct {
	S, T Vertex
}

func (e *basicEdge) Source() Vertex {
	return e.S
}

func (e *basicEdge) Target() Vertex {
	return e.T
}

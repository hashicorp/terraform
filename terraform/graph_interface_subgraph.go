package terraform

// GraphNodeSubPath says that a node is part of a graph with a
// different path, and the context should be adjusted accordingly.
type GraphNodeSubPath interface {
	Path() []string
}

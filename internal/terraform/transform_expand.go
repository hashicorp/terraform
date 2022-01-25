package terraform

// GraphNodeDynamicExpandable is an interface that nodes can implement
// to signal that they can be expanded at eval-time (hence dynamic).
// These nodes are given the eval context and are expected to return
// a new subgraph.
type GraphNodeDynamicExpandable interface {
	DynamicExpand(EvalContext) (*Graph, error)
}

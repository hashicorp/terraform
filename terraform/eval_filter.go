package terraform

// EvalNodeFilterFunc is the callback used to replace a node with
// another to node. To not do the replacement, just return the input node.
type EvalNodeFilterFunc func(EvalNode) EvalNode

// EvalNodeFilterable is an interface that can be implemented by
// EvalNodes to allow filtering of sub-elements. Note that this isn't
// a common thing to implement and you probably don't need it.
type EvalNodeFilterable interface {
	EvalNode
	Filter(EvalNodeFilterFunc)
}

// EvalFilter runs the filter on the given node and returns the
// final filtered value. This should be called rather than checking
// the EvalNode directly since this will properly handle EvalNodeFilterables.
func EvalFilter(node EvalNode, fn EvalNodeFilterFunc) EvalNode {
	if f, ok := node.(EvalNodeFilterable); ok {
		f.Filter(fn)
		return node
	}

	return fn(node)
}

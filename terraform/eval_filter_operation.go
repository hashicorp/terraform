package terraform

// EvalNodeOpFilterable is an interface that EvalNodes can implement
// to be filterable by the operation that is being run on Terraform.
type EvalNodeOpFilterable interface {
	IncludeInOp(walkOperation) bool
}

// EvalNodeFilterOp returns a filter function that filters nodes that
// include themselves in specific operations.
func EvalNodeFilterOp(op walkOperation) EvalNodeFilterFunc {
	return func(n EvalNode) EvalNode {
		include := true
		if of, ok := n.(EvalNodeOpFilterable); ok {
			include = of.IncludeInOp(op)
		}
		if include {
			return n
		}

		return EvalNoop{}
	}
}

// EvalOpFilter is an EvalNode implementation that is a proxy to
// another node but filters based on the operation.
type EvalOpFilter struct {
	// Ops is the list of operations to include this node in.
	Ops []walkOperation

	// Node is the node to execute
	Node EvalNode
}

// TODO: test
func (n *EvalOpFilter) Eval(ctx EvalContext) (interface{}, error) {
	return EvalRaw(n.Node, ctx)
}

// EvalNodeOpFilterable impl.
func (n *EvalOpFilter) IncludeInOp(op walkOperation) bool {
	for _, v := range n.Ops {
		if v == op {
			return true
		}
	}

	return false
}

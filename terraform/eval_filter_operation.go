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

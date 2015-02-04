package terraform

// EvalNode is the interface that must be implemented by graph nodes to
// evaluate/execute.
type EvalNode interface {
	// Args returns the arguments for this node as well as the list of
	// expected types. The expected types are only used for type checking
	// and not used at runtime.
	Args() ([]EvalNode, []EvalType)

	// Eval evaluates this node with the given context. The second parameter
	// are the argument values. These will match in order and 1-1 with the
	// results of the Args() return value.
	Eval(EvalContext, []interface{}) (interface{}, error)

	// Type returns the type that will be returned by this node.
	Type() EvalType
}

// GraphNodeEvalable is the interface that graph nodes must implement
// to enable valuation.
type GraphNodeEvalable interface {
	EvalTree() EvalNode
}

// Eval evaluates the given EvalNode with the given context, properly
// evaluating all args in the correct order.
func Eval(n EvalNode, ctx EvalContext) (interface{}, error) {
	argNodes, _ := n.Args()
	args := make([]interface{}, len(argNodes))
	for i, n := range argNodes {
		v, err := Eval(n, ctx)
		if err != nil {
			return nil, err
		}

		args[i] = v
	}

	return n.Eval(ctx, args)
}

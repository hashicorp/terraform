package terraform

import (
	"log"
)

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

// EvalEarlyExitError is a special error return value that can be returned
// by eval nodes that does an early exit.
type EvalEarlyExitError struct{}

func (EvalEarlyExitError) Error() string { return "early exit" }

// Eval evaluates the given EvalNode with the given context, properly
// evaluating all args in the correct order.
func Eval(n EvalNode, ctx EvalContext) (interface{}, error) {
	// Call the lower level eval which doesn't understand early exit,
	// and if we early exit, it isn't an error.
	result, err := eval(n, ctx)
	if err != nil {
		if _, ok := err.(EvalEarlyExitError); ok {
			return nil, nil
		}
	}

	return result, err
}

func eval(n EvalNode, ctx EvalContext) (interface{}, error) {
	argNodes, _ := n.Args()
	args := make([]interface{}, len(argNodes))
	for i, n := range argNodes {
		v, err := eval(n, ctx)
		if err != nil {
			return nil, err
		}

		args[i] = v
	}

	log.Printf("[DEBUG] eval: %T", n)
	output, err := n.Eval(ctx, args)
	if err != nil {
		log.Printf("[ERROR] eval: %T, err: %s", n, err)
	}

	return output, err
}

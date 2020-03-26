package terraform

import (
	"log"

	"github.com/hashicorp/terraform/tfdiags"
)

// EvalNode is the interface that must be implemented by graph nodes to
// evaluate/execute.
type EvalNode interface {
	// Eval evaluates this node with the given context. The second parameter
	// are the argument values. These will match in order and 1-1 with the
	// results of the Args() return value.
	Eval(EvalContext) (interface{}, error)
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
	result, err := EvalRaw(n, ctx)
	if err != nil {
		if _, ok := err.(EvalEarlyExitError); ok {
			return nil, nil
		}
	}

	return result, err
}

// EvalRaw is like Eval except that it returns all errors, even if they
// signal something normal such as EvalEarlyExitError.
func EvalRaw(n EvalNode, ctx EvalContext) (interface{}, error) {
	log.Printf("[TRACE] eval: %T", n)
	output, err := n.Eval(ctx)
	if err != nil {
		switch err.(type) {
		case EvalEarlyExitError:
			log.Printf("[TRACE] eval: %T, early exit err: %s", n, err)
		case tfdiags.NonFatalError:
			log.Printf("[WARN] eval: %T, non-fatal err: %s", n, err)
		default:
			log.Printf("[ERROR] eval: %T, err: %s", n, err)
		}
	}

	return output, err
}

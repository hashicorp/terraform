package terraform

import (
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalSequence is an EvalNode that evaluates in sequence.
type EvalSequence struct {
	Nodes []EvalNode
}

func (n *EvalSequence) Eval(ctx EvalContext) (interface{}, error) {
	var diags tfdiags.Diagnostics

	for _, n := range n.Nodes {
		if n == nil {
			continue
		}

		if _, err := EvalRaw(n, ctx); err != nil {
			if _, isEarlyExit := err.(EvalEarlyExitError); isEarlyExit {
				// In this path we abort early, losing any non-error
				// diagnostics we saw earlier.
				return nil, err
			}
			diags = diags.Append(err)
			if diags.HasErrors() {
				// Halt if we get some errors, but warnings are okay.
				break
			}
		}
	}

	return nil, diags.ErrWithWarnings()
}

// EvalNodeFilterable impl.
func (n *EvalSequence) Filter(fn EvalNodeFilterFunc) {
	for i, node := range n.Nodes {
		n.Nodes[i] = fn(node)
	}
}

package terraform

import "github.com/hashicorp/terraform/tfdiags"

// GraphNodeExecutable is the interface that graph nodes must implement to
// enable execution. This is an alternative to GraphNodeEvalable, which is in
// the process of being removed. A given graph node should _not_ implement both
// GraphNodeExecutable and GraphNodeEvalable.
type GraphNodeExecutable interface {
	Execute(EvalContext, walkOperation) tfdiags.Diagnostics
}

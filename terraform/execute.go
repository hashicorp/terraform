// like eval, but execute!
package terraform

import (
	"github.com/hashicorp/terraform/tfdiags"
)

// GraphNodeEvalable is the interface that graph nodes must implement
// to enable valuation.
type GraphNodeExecutable interface {
	Execute(EvalContext) tfdiags.Diagnostics
}

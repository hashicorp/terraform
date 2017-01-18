package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// EvalCountFixComputed is an EvalNode that checks if a resource count
// is computed and try to fix the value.
// It can only be fixed if the walk operation is either walkInput or walkValidate
// Otherwise it errors
type EvalCountFixComputed struct {
	Resource *config.Resource
}

// TODO: test
func (n *EvalCountFixComputed) Eval(ctx EvalContext) (interface{}, error) {
	if ctx.CanIgnoreMissingCountExpansion() {
		n.Resource.DeferCountComputation = true
	} else {
		n.Resource.DeferCountComputation = false

		if n.Resource.RawCount.Value() == unknownValue() {
			return nil, fmt.Errorf(
				"%s: value of 'count' cannot be computed",
				n.Resource.Id())
		}
	}

	return nil, nil
}

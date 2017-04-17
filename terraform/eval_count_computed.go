package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// EvalCountCheckComputed is an EvalNode that checks if a resource count
// is computed and errors if so. This can possibly happen across a
// module boundary and we don't yet support this.
type EvalCountCheckComputed struct {
	Resource *config.Resource
}

// TODO: test
func (n *EvalCountCheckComputed) Eval(ctx EvalContext) (interface{}, error) {
	if n.Resource.RawCount.Value() == unknownValue() {
		return nil, fmt.Errorf(
			"%s: value of 'count' cannot be computed",
			n.Resource.Id())
	}

	return nil, nil
}

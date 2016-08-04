package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// EvalPreventDestroy is an EvalNode implementation that returns an
// error if a resource has PreventDestroy configured and the diff
// would destroy the resource.
type EvalCheckPreventDestroy struct {
	Resource *config.Resource
	Diff     **InstanceDiff
}

func (n *EvalCheckPreventDestroy) Eval(ctx EvalContext) (interface{}, error) {
	if n.Diff == nil || *n.Diff == nil || n.Resource == nil {
		return nil, nil
	}

	diff := *n.Diff
	preventDestroy := n.Resource.Lifecycle.PreventDestroy

	if diff.GetDestroy() && preventDestroy {
		return nil, fmt.Errorf(preventDestroyErrStr, n.Resource.Id())
	}

	return nil, nil
}

const preventDestroyErrStr = `%s: the plan would destroy this resource, but it currently has lifecycle.prevent_destroy set to true. To avoid this error and continue with the plan, either disable lifecycle.prevent_destroy or adjust the scope of the plan using the -target flag.`

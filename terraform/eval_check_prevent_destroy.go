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

	if diff.Destroy && preventDestroy {
		return nil, fmt.Errorf(preventDestroyErrStr, n.Resource.Id())
	}

	return nil, nil
}

const preventDestroyErrStr = `%s: plan would destroy, but resource has prevent_destroy set. To avoid this error, either disable prevent_destroy, or change your config so the plan does not destroy this resource.`

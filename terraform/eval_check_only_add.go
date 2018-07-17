package terraform

import (
	"github.com/hashicorp/terraform/config"
)

// EvalOnlyAdd is an EvalNode implementation that turns off the destroy flag if it is ever set
type EvalCheckOnlyAdd struct {
	Resource   *config.Resource
	ResourceId string
	Diff       **InstanceDiff
}

func (n *EvalCheckOnlyAdd) Eval(ctx EvalContext) (interface{}, error) {
	if n.Diff == nil || *n.Diff == nil || n.Resource == nil {
		return nil, nil
	}

	diff := *n.Diff
	onlyAdd := n.Resource.Lifecycle.OnlyAdd

	if diff.GetDestroy() && onlyAdd {
		diff.SetDestroy(false)
	}

	return nil, nil
}

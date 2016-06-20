package terraform

import (
	"strings"

	"github.com/hashicorp/terraform/config"
)

// EvalHideIgnored is an EvalNode implementation that removes ignored
// attributes from the state.
type EvalHideIgnored struct {
	Resource      *config.Resource
	State         **InstanceState
}

func (n *EvalHideIgnored) Eval(ctx EvalContext) (interface{}, error) {
	if n.State == nil || *n.State == nil || n.Resource == nil || n.Resource.Id() == "" {
		return nil, nil
	}
	var state *InstanceState = *n.State

	if !n.Resource.Lifecycle.HideIgnored {
		return nil, nil
	}

	ignoreChanges := n.Resource.Lifecycle.IgnoreChanges
	if len(ignoreChanges) == 0 {
		return nil, nil
	}

	for _, ignoredName := range ignoreChanges {
		for name := range state.Attributes {
			if strings.HasPrefix(name, ignoredName) {
				delete(state.Attributes, name)
			}
		}
	}

	return nil, nil
}

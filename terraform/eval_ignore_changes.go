package terraform

import (
	"strings"

	"github.com/hashicorp/terraform/config"
)

// EvalIgnoreChanges is an EvalNode implementation that removes diff
// attributes if their name matches names provided by the resource's
// IgnoreChanges lifecycle.
type EvalIgnoreChanges struct {
	Resource *config.Resource
	Diff     **InstanceDiff
}

func (n *EvalIgnoreChanges) Eval(ctx EvalContext) (interface{}, error) {
	if n.Diff == nil || *n.Diff == nil || n.Resource == nil || n.Resource.Id() == "" {
		return nil, nil
	}

	diff := *n.Diff
	ignoreChanges := n.Resource.Lifecycle.IgnoreChanges

	for _, ignoredName := range ignoreChanges {
		for name := range diff.Attributes {
			if strings.HasPrefix(name, ignoredName) {
				delete(diff.Attributes, name)
			}
		}
	}

	return nil, nil
}

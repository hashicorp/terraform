// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

type ActionInvokeTransformer struct {
	Config        *configs.Config
	ActionTargets []addrs.Targetable
	Operation     walkOperation

	queryPlanMode bool
}

func (t *ActionInvokeTransformer) Transform(g *Graph) error {
	if t.Operation != walkPlan || t.queryPlanMode || len(t.ActionTargets) == 0 {
		return nil
	}

	// Then we're invoking and we're just going to include the actions that
	// have been specifically asked for.
	for _, target := range t.ActionTargets {
		var config *configs.Action
		switch target := target.(type) {
		case addrs.AbsAction:
			module := t.Config.DescendantForInstance(target.Module)
			if module != nil {
				config = module.Module.Actions[target.Action.String()]
			}
		case addrs.AbsActionInstance:
			module := t.Config.DescendantForInstance(target.Module)
			if module != nil {
				config = module.Module.Actions[target.Action.Action.String()]
			}
		}

		if config == nil {
			return fmt.Errorf("action %s does not exist in the configuration", target.String())
		}

		g.Add(&nodeActionInvokeExpand{
			Target: target,
			Config: config,
		})
	}

	return nil

}

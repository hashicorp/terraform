// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
)

type ActionInvokeApplyTransformer struct {
	Config        *configs.Config
	ActionTargets []addrs.Targetable
	Operation     walkOperation
	Changes       *plans.ChangesSrc

	queryPlanMode bool
}

func (t *ActionInvokeApplyTransformer) Transform(g *Graph) error {
	if t.Operation != walkApply || t.queryPlanMode || len(t.ActionTargets) == 0 {
		return nil
	}

	// We just want to add all invoke triggered action invocations
	for _, action := range t.Changes.ActionInvocations {
		// get the config for the action!
		cfg := t.Config.DescendantForInstance(action.Addr.Module)
		actionCfg := cfg.Module.ActionByAddr(action.Addr.Action.Action)

		// Add nodes for each action invocation
		node := &nodeActionTriggerApplyInstance{
			ActionInvocation: action,
			actionConfig:     actionCfg,
		}
		g.Add(node)
	}

	return nil
}

// Copyright (c) HashiCorp, Inc.
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
		// Add nodes for each action invocation
		node := &nodeActionTriggerApplyInstance{
			ActionInvocation: action,
		}
		g.Add(node)
	}

	return nil
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeActionInvoke struct {
	Target addrs.Targetable
}

var (
	_ GraphNodeExecutable = (*nodeActionInvoke)(nil)
	//_ GraphNodeReferencer      = (*nodeActionInvoke)(nil)
	//_ dag.GraphNodeDotter      = (*nodeActionInvoke)(nil)
	//_ GraphNodeActionProviders = (*nodeActionInvoke)(nil)
)

func (n *nodeActionInvoke) Name() string {
	return n.Target.String()
}

func (n *nodeActionInvoke) DotNode(string, *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: n.Name(),
	}
}

func (n *nodeActionInvoke) Execute(ctx EvalContext, wo walkOperation) (diags tfdiags.Diagnostics) {
	aaiAddr, ok := n.Target.(addrs.AbsActionInstance)
	if !ok {
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Hey error",
			"Hey details",
		))
	}

	ai, ok := ctx.Actions().GetActionInstance(aaiAddr)
	if !ok {
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Action instance not found",
			"Action instance not found",
		))
	}

	provider, _, err := getProvider(ctx, ai.ProviderAddr)
	if err != nil {
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Hey error3",
			"Hey details3",
		))
	}

	switch wo {
	case walkPlan:
		res := provider.PlanAction(providers.PlanActionRequest{
			ActionType:         aaiAddr.Action.Action.Type,
			ProposedActionData: ai.ConfigValue,
			LinkedResources:    nil,
			ClientCapabilities: providers.ClientCapabilities{},
		})

		if res.Diagnostics.HasErrors() {
			return diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Hey error4",
				"Hey details4",
			))
		}

		ctx.Changes().AppendActionInvocation(&plans.ActionInvocationInstance{
			Addr:                    aaiAddr,
			TriggeringResourceAddr:  addrs.AbsResourceInstance{},
			TriggerEvent:            0,
			ActionTriggerBlockIndex: 0,
			ActionsListIndex:        0,
			ProviderAddr:            ai.ProviderAddr,
		})
	case walkApply:
		res := provider.InvokeAction(providers.InvokeActionRequest{
			ActionType:        aaiAddr.Action.Action.Type,
			PlannedActionData: ai.ConfigValue,
			LinkedResources:   nil,
		})

		if res.Diagnostics.HasErrors() {
			return diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Error while invoking provider action",
				"---TODO---",
			))
		}
	default:
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid walk operation",
			fmt.Sprintf("Invalid walk operation: %s", wo),
		))
	}

	return nil
}

func (n *nodeActionInvoke) ModulePath() addrs.Module {
	aai, ok := n.Target.(addrs.AbsActionInstance)
	if !ok {
		panic("not an abs action instance")
	}
	return aai.Module.Module()
}

func (n *nodeActionInvoke) References() []*addrs.Reference {
	aai, ok := n.Target.(addrs.AbsActionInstance)
	if !ok {
		panic("not an abs action instance")
	}

	var refs []*addrs.Reference
	refs = append(refs, &addrs.Reference{
		Subject: aai.Action,
	})

	return refs
}

func (n *nodeActionInvoke) Actions() []addrs.ConfigAction {
	aai, ok := n.Target.(addrs.AbsActionInstance)
	if !ok {
		panic("not an abs action instance")
	}

	return []addrs.ConfigAction{aai.ConfigAction()}
}

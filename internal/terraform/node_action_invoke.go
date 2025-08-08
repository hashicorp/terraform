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
	Target addrs.AbsActionInstance
}

var (
	_ GraphNodeExecutable = (*nodeActionInvoke)(nil)
	_ GraphNodeReferencer = (*nodeActionInvoke)(nil)
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
	ai, ok := ctx.Actions().GetActionInstance(n.Target)
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
			"Provider not found",
			"Provider not found",
		))
	}

	switch wo {
	case walkPlan:
		resp := provider.PlanAction(providers.PlanActionRequest{
			ActionType:         n.Target.Action.Action.Type,
			ProposedActionData: ai.ConfigValue,
			LinkedResources:    nil,
			ClientCapabilities: providers.ClientCapabilities{},
		})

		if resp.Diagnostics.HasErrors() {
			return diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Hey error4",
				"Hey details4",
			))
		}

		ctx.Changes().AppendActionInvocation(&plans.ActionInvocationInstance{
			Addr:          n.Target,
			ActionTrigger: plans.InvokeCmdActionTrigger{},
			ProviderAddr:  ai.ProviderAddr,
		})
	case walkApply:
		resp := provider.InvokeAction(providers.InvokeActionRequest{
			ActionType:        n.Target.Action.Action.Type,
			PlannedActionData: ai.ConfigValue,
			LinkedResources:   nil,
		})

		if resp.Diagnostics.HasErrors() {
			return diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Error while invoking provider action",
				"---TODO---",
			))
		}

		hookIdentity := HookActionIdentity{
			Addr:          n.Target,
			ActionTrigger: plans.InvokeCmdActionTrigger{},
		}

		for event := range resp.Events {
			switch ev := event.(type) {
			case providers.InvokeActionEvent_Progress:
				ctx.Hook(func(h Hook) (HookAction, error) {
					return h.ProgressAction(hookIdentity, ev.Message)
				})
			case providers.InvokeActionEvent_Completed:
				diags = diags.Append(ev.Diagnostics)
				ctx.Hook(func(h Hook) (HookAction, error) {
					return h.CompleteAction(hookIdentity, ev.Diagnostics.Err())
				})
				if ev.Diagnostics.HasErrors() {
					// TODO: We would want to add some warning / error telling the user how to recover
					// from this, or maybe attach this info to the diagnostics sent by the provider.
					// For now we just return the diagnostics.

					return diags
				}
			default:
				panic(fmt.Sprintf("unexpected action event type %T", ev))
			}
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
	return n.Target.Module.Module()
}

func (n *nodeActionInvoke) References() []*addrs.Reference {
	var refs []*addrs.Reference
	refs = append(refs, &addrs.Reference{
		Subject: n.Target.Action,
	})

	return refs
}

func (n *nodeActionInvoke) Actions() []addrs.ConfigAction {
	return []addrs.ConfigAction{n.Target.ConfigAction()}
}

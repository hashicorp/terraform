// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/actions"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeActionApply struct {
	TriggeringResourceaddrs addrs.AbsResourceInstance
	ActionInvocations       []*plans.ActionInvocationInstance
}

var (
	_ GraphNodeExecutable      = (*nodeActionApply)(nil)
	_ GraphNodeReferencer      = (*nodeActionApply)(nil)
	_ dag.GraphNodeDotter      = (*nodeActionApply)(nil)
	_ GraphNodeActionProviders = (*nodeActionApply)(nil)
)

func (n *nodeActionApply) Name() string {
	return fmt.Sprintf("%s after actions", n.TriggeringResourceaddrs)
}

func (n *nodeActionApply) DotNode(string, *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: n.Name(),
	}
}

func (n *nodeActionApply) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
	finishedActionInvocations, diags := invokeActions(ctx, n.ActionInvocations)
	if diags.HasErrors() {
		// Since the actions running after the resource apply have failed, we want to give the user
		// directions on how to recover from this. In this case we want to give the user a list of
		// actions to re-run using the `terraform invoke` command.

		// Since the actions are run in order, we can remove however many actions we have already
		// run from the list of action invocations.
		numberOfMissingActions := len(n.ActionInvocations) - len(finishedActionInvocations)
		// This should always be the case, but we check just to be sure.
		if numberOfMissingActions > 0 {
			// We want to give the user a list of actions to re-run.
			actionsToReRun := []string{}
			for _, invocation := range n.ActionInvocations[:numberOfMissingActions] {
				actionsToReRun = append(actionsToReRun, fmt.Sprintf("  - %s", invocation.Addr.String()))
			}

			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Actions failed for %s", n.TriggeringResourceaddrs),
				fmt.Sprintf(
					"The following actions failed: \n%s\nYou can re-run them using the `terraform invoke <action address>` command.",
					strings.Join(actionsToReRun, "\n"),
				),
			))
		}
	}
	return diags
}

func invokeActions(ctx EvalContext, actionInvocations []*plans.ActionInvocationInstance) ([]*plans.ActionInvocationInstance, tfdiags.Diagnostics) {
	var finishedActionInvocations []*plans.ActionInvocationInstance
	var diags tfdiags.Diagnostics
	// First we order the action invocations by their trigger block index and events list index.
	// This way we have the correct order of execution.
	orderedActionInvocations := make([]*plans.ActionInvocationInstance, len(actionInvocations))
	copy(orderedActionInvocations, actionInvocations)
	sort.Slice(orderedActionInvocations, func(i, j int) bool {
		if orderedActionInvocations[i].ActionTriggerBlockIndex == orderedActionInvocations[j].ActionTriggerBlockIndex {
			return orderedActionInvocations[i].ActionsListIndex < orderedActionInvocations[j].ActionsListIndex
		}
		return orderedActionInvocations[i].ActionTriggerBlockIndex < orderedActionInvocations[j].ActionTriggerBlockIndex
	})

	// Now we ensure we have an expanded action instance for each action invocations.
	orderedActionData := make([]*actions.ActionData, len(orderedActionInvocations))
	for i, invocation := range orderedActionInvocations {
		ai, ok := ctx.Actions().GetActionInstance(invocation.Addr)
		if !ok {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Action instance not found",
				"Could not find action instance for address "+invocation.Addr.String(),
			))
			return finishedActionInvocations, diags
		}

		orderedActionData[i] = ai
	}

	// Now we have everything in place to execute the actions in the correct order.
	// TODO: Handle verifying the condition here, if we have any.

	// We run every action sequentially, as the order of execution is important. We also abort if
	// an action fails, as we don't want to continue executing actions or nodes that depend on it.

	for i, actionData := range orderedActionData {
		ai := orderedActionInvocations[i]
		provider, _, err := getProvider(ctx, actionData.ProviderAddr)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Failed to get provider for %s", ai.Addr),
				fmt.Sprintf("Failed to get provider: %s", err),
			))
			return finishedActionInvocations, diags
		}

		// We don't want to send the marks, but all marks are okay in the context of an action invocation.
		unmarkedConfigValue, _ := actionData.ConfigValue.UnmarkDeep()

		hookIdentity := HookActionIdentity{
			Addr:                    ai.Addr,
			TriggeringResourceAddr:  ai.TriggeringResourceAddr,
			ActionTriggerBlockIndex: ai.ActionTriggerBlockIndex,
			ActionsListIndex:        ai.ActionsListIndex,
		}

		ctx.Hook(func(h Hook) (HookAction, error) {
			return h.StartAction(hookIdentity)
		})
		resp := provider.InvokeAction(providers.InvokeActionRequest{
			ActionType:        orderedActionInvocations[i].Addr.Action.Action.Type,
			PlannedActionData: unmarkedConfigValue,
		})

		diags = diags.Append(resp.Diagnostics)
		if resp.Diagnostics.HasErrors() {
			return finishedActionInvocations, diags
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

					return finishedActionInvocations, diags
				} else {
					finishedActionInvocations = append(finishedActionInvocations, ai)
				}
			default:
				panic(fmt.Sprintf("unexpected action event type %T", ev))
			}
		}
	}

	return finishedActionInvocations, diags
}

func (n *nodeActionApply) ModulePath() addrs.Module {
	return n.TriggeringResourceaddrs.Module.Module()
}

func (n *nodeActionApply) References() []*addrs.Reference {
	var refs []*addrs.Reference

	// We reference each action instance that we are going to execute.
	for _, invocation := range n.ActionInvocations {
		refs = append(refs, &addrs.Reference{
			Subject: invocation.Addr.Action,
		})
	}

	return refs
}

func (n *nodeActionApply) ActionProviders() []addrs.AbsProviderConfig {
	ret := []addrs.AbsProviderConfig{}
	for _, invocation := range n.ActionInvocations {
		ret = append(ret, invocation.ProviderAddr)
	}
	return ret
}

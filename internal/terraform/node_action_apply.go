// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/actions"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeActionApply struct {
	TriggeringResourceaddrs addrs.AbsResourceInstance
	ActionInvocations       []*plans.ActionInvocationInstanceSrc
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

func (n *nodeActionApply) Execute(ctx EvalContext, _ walkOperation) (diags tfdiags.Diagnostics) {
	return invokeActions(ctx, n.ActionInvocations)
}

func invokeActions(ctx EvalContext, actionInvocations []*plans.ActionInvocationInstanceSrc) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	// First we order the action invocations by their trigger block index and events list index.
	// This way we have the correct order of execution.
	orderedActionInvocations := make([]*plans.ActionInvocationInstance, 0, len(actionInvocations))
	for _, invocation := range actionInvocations {
		ai := ctx.Changes().GetActionInvocation(invocation.Addr, invocation.TriggeringResourceAddr, invocation.ActionTriggerBlockIndex, invocation.ActionsListIndex)

		if ai == nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Failed to find action invocation instance %s in changes.", ai.Addr),
				fmt.Sprintf("The action invocation instance %s was not found in the changes for %s.", ai.Addr, ai.TriggeringResourceAddr.String()),
			))
			return diags
		}

		orderedActionInvocations = append(orderedActionInvocations, ai)
	}
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
			return diags
		}

		orderedActionData[i] = ai
	}

	// Now we have everything in place to execute the actions in the correct order.
	// TODO: Handle verifying the condition here, if we have any.

	// We run every action sequentially, as the order of execution is important. We also abort if
	// an action fails, as we don't want to continue executing actions or nodes that depend on it.

	for i, actionData := range orderedActionData {
		ai := orderedActionInvocations[i]
		provider, schema, err := getProvider(ctx, actionData.ProviderAddr)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Failed to get provider for %s", ai.Addr),
				fmt.Sprintf("Failed to get provider: %s", err),
			))
			return diags
		}

		actionSchema, ok := schema.Actions[ai.Addr.Action.Action.Type]
		if !ok {
			// This should have been caught earlier
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Action %s not found in provider schema", ai.Addr),
				fmt.Sprintf("The action %s was not found in the provider schema for %s", ai.Addr.Action.Action.Type, actionData.ProviderAddr),
			))
			return diags
		}

		// We don't want to send the marks, but all marks are okay in the context of an action invocation.
		unmarkedConfigValue, _ := actionData.ConfigValue.UnmarkDeep()

		// Validate that what we planned matches the action data we have.
		errs := objchange.AssertObjectCompatible(actionSchema.ConfigSchema, ai.ConfigValue, unmarkedConfigValue)
		for _, err := range errs {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider produced inconsistent final plan",
				fmt.Sprintf(
					"When expanding the plan for %s to include new values learned so far during apply, provider %q produced an invalid new value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
					ai.Addr, actionData.ProviderAddr.Provider.String(), tfdiags.FormatError(err),
				),
			))
		}

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
			return diags
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
	}

	return diags
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

func (n *nodeActionApply) ActionProviders() []addrs.ProviderConfig {
	ret := []addrs.ProviderConfig{}
	for _, invocation := range n.ActionInvocations {
		ret = append(ret, invocation.ProviderAddr)
	}
	return ret
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeActionTriggerApply struct {
	ActionInvocation *plans.ActionInvocationInstanceSrc
	resolvedProvider addrs.AbsProviderConfig
}

var (
	_ GraphNodeExecutable = (*nodeActionTriggerApply)(nil)
	_ GraphNodeReferencer = (*nodeActionTriggerApply)(nil)
)

func (n *nodeActionTriggerApply) Name() string {
	return "action_apply_" + n.ActionInvocation.Addr.String()
}

func (n *nodeActionTriggerApply) Execute(ctx EvalContext, wo walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	actionInvocation := n.ActionInvocation

	// TODO: Handle verifying the condition here, if we have any.
	ai := ctx.Changes().GetActionInvocation(actionInvocation.Addr, actionInvocation.ActionTrigger)
	if ai == nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Action invocation not found in plan",
			"Could not find action invocation for address "+actionInvocation.Addr.String(),
		))
		return diags
	}
	actionData, ok := ctx.Actions().GetActionInstance(ai.Addr)
	if !ok {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Action instance not found",
			"Could not find action instance for address "+ai.Addr.String(),
		))
		return diags
	}
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
		Addr:          ai.Addr,
		ActionTrigger: ai.ActionTrigger,
	}

	ctx.Hook(func(h Hook) (HookAction, error) {
		return h.StartAction(hookIdentity)
	})
	resp := provider.InvokeAction(providers.InvokeActionRequest{
		ActionType:         ai.Addr.Action.Action.Type,
		PlannedActionData:  unmarkedConfigValue,
		ClientCapabilities: ctx.ClientCapabilities(),
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
				return diags
			}
		default:
			panic(fmt.Sprintf("unexpected action event type %T", ev))
		}
	}

	return diags
}

func (n *nodeActionTriggerApply) ProvidedBy() (addr addrs.ProviderConfig, exact bool) {
	return n.ActionInvocation.ProviderAddr, true

}

func (n *nodeActionTriggerApply) Provider() (provider addrs.Provider) {
	return n.ActionInvocation.ProviderAddr.Provider
}

func (n *nodeActionTriggerApply) SetProvider(config addrs.AbsProviderConfig) {
	n.resolvedProvider = config
}

func (n *nodeActionTriggerApply) References() []*addrs.Reference {
	var refs []*addrs.Reference

	refs = append(refs, &addrs.Reference{
		Subject: n.ActionInvocation.Addr.Action.Action,
	})

	return refs
}

// GraphNodeModulePath
func (n *nodeActionTriggerApply) ModulePath() addrs.Module {
	return addrs.RootModule
}

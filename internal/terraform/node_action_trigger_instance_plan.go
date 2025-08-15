// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeActionTriggerPlanInstance struct {
	actionAddress    addrs.AbsActionInstance
	resolvedProvider addrs.AbsProviderConfig
	actionConfig     *configs.Action

	lifecycleActionTrigger *lifecycleActionTriggerInstance
}

type lifecycleActionTriggerInstance struct {
	resourceAddress addrs.AbsResourceInstance
	events          []configs.ActionTriggerEvent
	//condition       hcl.Expression
	actionTriggerBlockIndex int
	actionListIndex         int
	invokingSubject         *hcl.Range
}

func (at *lifecycleActionTriggerInstance) Name() string {
	return fmt.Sprintf("%s.lifecycle.action_trigger[%d].actions[%d]", at.resourceAddress.String(), at.actionTriggerBlockIndex, at.actionListIndex)
}

var (
	_ GraphNodeModuleInstance = (*nodeActionTriggerPlanInstance)(nil)
	_ GraphNodeExecutable     = (*nodeActionTriggerPlanInstance)(nil)
)

func (n *nodeActionTriggerPlanInstance) Name() string {
	triggeredBy := "triggered by "
	if n.lifecycleActionTrigger != nil {
		triggeredBy += n.lifecycleActionTrigger.resourceAddress.String()
	} else {
		triggeredBy += "unknown"
	}

	return fmt.Sprintf("%s %s", n.actionAddress.String(), triggeredBy)
}

func (n *nodeActionTriggerPlanInstance) Execute(ctx EvalContext, operation walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if n.lifecycleActionTrigger == nil {
		panic("Only actions triggered by plan and apply are supported")
	}

	change := ctx.Changes().GetResourceInstanceChange(n.lifecycleActionTrigger.resourceAddress, n.lifecycleActionTrigger.resourceAddress.CurrentObject().DeposedKey)
	if change == nil {
		panic("change cannot be nil")
	}
	triggeringEvent, isTriggered := actionIsTriggeredByEvent(n.lifecycleActionTrigger.events, change.Action)
	if !isTriggered {
		return diags
	}
	if triggeringEvent == nil {
		panic("triggeringEvent cannot be nil")
	}

	actionInstance, ok := ctx.Actions().GetActionInstance(n.actionAddress)

	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to non-existant action instance",
			Detail:   "Action instance was not found in the current context.",
			Subject:  n.lifecycleActionTrigger.invokingSubject,
		})
		return diags
	}

	provider, _, err := getProvider(ctx, actionInstance.ProviderAddr)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to get provider",
			Detail:   fmt.Sprintf("Failed to get provider: %s", err),
			Subject:  n.lifecycleActionTrigger.invokingSubject,
		})

		return diags
	}

	resp := provider.PlanAction(providers.PlanActionRequest{
		ActionType:         n.actionAddress.Action.Action.Type,
		ProposedActionData: actionInstance.ConfigValue,
		ClientCapabilities: ctx.ClientCapabilities(),
	})

	// TODO: Deal with deferred responses
	diags = diags.Append(resp.Diagnostics)
	if diags.HasErrors() {
		return diags
	}

	ctx.Changes().AppendActionInvocation(&plans.ActionInvocationInstance{
		Addr:         n.actionAddress,
		ProviderAddr: actionInstance.ProviderAddr,
		ActionTrigger: plans.LifecycleActionTrigger{
			TriggeringResourceAddr:  n.lifecycleActionTrigger.resourceAddress,
			ActionTriggerEvent:      *triggeringEvent,
			ActionTriggerBlockIndex: n.lifecycleActionTrigger.actionTriggerBlockIndex,
			ActionsListIndex:        n.lifecycleActionTrigger.actionListIndex,
		},
		ConfigValue: actionInstance.ConfigValue,
	})

	return diags
}

func (n *nodeActionTriggerPlanInstance) ModulePath() addrs.Module {
	return n.Path().Module()
}

func (n *nodeActionTriggerPlanInstance) Path() addrs.ModuleInstance {
	// Actions can only be triggered by the CLI in which case they belong to the module they are in
	// or by resources during plan/apply in which case both the resource and action must belong
	// to the same module. So we can simply return the module path of the action.
	return n.actionAddress.Module
}

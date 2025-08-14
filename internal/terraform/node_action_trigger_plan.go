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

type nodeActionTriggerPlan struct {
	actionAddress    addrs.AbsActionInstance
	resolvedProvider addrs.AbsProviderConfig
	actionConfig     *configs.Action

	lifecycleActionTrigger *lifecycleActionTrigger
}

type lifecycleActionTrigger struct {
	resourceAddress addrs.ConfigResource
	events          []configs.ActionTriggerEvent
	//condition       hcl.Expression
	actionTriggerBlockIndex int
	actionListIndex         int
	invokingSubject         *hcl.Range
}

func (at *lifecycleActionTrigger) Name() string {
	return fmt.Sprintf("%s.lifecycle.action_trigger[%d].actions[%d]", at.resourceAddress.String(), at.actionTriggerBlockIndex, at.actionListIndex)
}

var (
	_ GraphNodeExecutable       = (*nodeActionTriggerPlan)(nil)
	_ GraphNodeReferencer       = (*nodeActionTriggerPlan)(nil)
	_ GraphNodeProviderConsumer = (*nodeActionTriggerPlan)(nil)
)

func (n *nodeActionTriggerPlan) Name() string {
	triggeredBy := "triggered by "
	if n.lifecycleActionTrigger != nil {
		triggeredBy += n.lifecycleActionTrigger.resourceAddress.String()
	} else {
		triggeredBy += "unknown"
	}

	return fmt.Sprintf("%s %s", n.actionAddress.String(), triggeredBy)
}

func (n *nodeActionTriggerPlan) Execute(ctx EvalContext, operation walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if n.lifecycleActionTrigger == nil {
		panic("Only actions triggered by plan and apply are supported")
	}

	_, keys, _ := ctx.InstanceExpander().ResourceInstanceKeys(n.lifecycleActionTrigger.resourceAddress.Absolute(addrs.RootModuleInstance))
	for _, key := range keys {
		change := ctx.Changes().
			GetResourceInstanceChange(
				n.lifecycleActionTrigger.resourceAddress.Absolute(
					addrs.RootModuleInstance).
					Instance(key),
				addrs.NotDeposed)
		if change == nil {
			panic("change cannot be nil")
		}
		triggeringEvent, isTriggered := actionIsTriggeredByEvent(n.lifecycleActionTrigger.events, change.Action)
		if !isTriggered {
			return nil
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
				TriggeringResourceAddr:  n.lifecycleActionTrigger.resourceAddress.Absolute(addrs.RootModuleInstance).Instance(key),
				ActionTriggerEvent:      *triggeringEvent,
				ActionTriggerBlockIndex: n.lifecycleActionTrigger.actionTriggerBlockIndex,
				ActionsListIndex:        n.lifecycleActionTrigger.actionListIndex,
			},
			ConfigValue: actionInstance.ConfigValue,
		})

	}

	return diags
}

func (n *nodeActionTriggerPlan) ModulePath() addrs.Module {
	return addrs.RootModule
}

func (n *nodeActionTriggerPlan) References() []*addrs.Reference {
	var refs []*addrs.Reference
	refs = append(refs, &addrs.Reference{
		Subject: n.actionAddress.Action,
	})

	if n.lifecycleActionTrigger != nil {
		refs = append(refs, &addrs.Reference{
			Subject: n.lifecycleActionTrigger.resourceAddress.Resource,
		})
	}

	return refs
}

func (n *nodeActionTriggerPlan) ProvidedBy() (addr addrs.ProviderConfig, exact bool) {
	if n.resolvedProvider.Provider.Type != "" {
		return n.resolvedProvider, true
	}

	// Since we always have a config, we can use it
	relAddr := n.actionConfig.ProviderConfigAddr()
	return addrs.LocalProviderConfig{
		LocalName: relAddr.LocalName,
		Alias:     relAddr.Alias,
	}, false
}

func (n *nodeActionTriggerPlan) Provider() (provider addrs.Provider) {
	return n.actionConfig.Provider
}

func (n *nodeActionTriggerPlan) SetProvider(config addrs.AbsProviderConfig) {
	n.resolvedProvider = config
}

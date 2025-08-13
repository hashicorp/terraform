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
	resourceAddress  addrs.ConfigResource
	actionAddress    addrs.AbsActionInstance
	events           []configs.ActionTriggerEvent
	resolvedProvider addrs.AbsProviderConfig
	//condition       hcl.Expression
	actionConfig            *configs.Action
	actionTriggerBlockIndex int
	actionListIndex         int
	invokingSubject         *hcl.Range
}

var (
	_ GraphNodeExecutable       = (*nodeActionTriggerPlan)(nil)
	_ GraphNodeReferencer       = (*nodeActionTriggerPlan)(nil)
	_ GraphNodeProviderConsumer = (*nodeActionTriggerPlan)(nil)
)

func (n *nodeActionTriggerPlan) Name() string {
	return "action_" + n.resourceAddress.String()
}

func (n *nodeActionTriggerPlan) Execute(ctx EvalContext, operation walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	_, keys, _ := ctx.InstanceExpander().ResourceInstanceKeys(n.resourceAddress.Absolute(addrs.RootModuleInstance))
	for _, key := range keys {
		change := ctx.Changes().
			GetResourceInstanceChange(
				n.resourceAddress.Absolute(
					addrs.RootModuleInstance).
					Instance(key),
				addrs.NotDeposed)
		if change == nil {
			panic("change cannot be nil")
		}
		triggeringEvent, isTriggered := actionIsTriggeredByEvent(n.events, change.Action)
		if !isTriggered {
			return nil
		}

		actionInstance, ok := ctx.Actions().GetActionInstance(n.actionAddress)

		if !ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reference to non-existant action instance",
				Detail:   "Action instance was not found in the current context.",
				Subject:  n.invokingSubject,
			})
			return diags
		}

		provider, _, err := getProvider(ctx, actionInstance.ProviderAddr)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Failed to get provider",
				Detail:   fmt.Sprintf("Failed to get provider: %s", err),
				Subject:  n.invokingSubject,
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
				TriggeringResourceAddr:  n.resourceAddress.Absolute(addrs.RootModuleInstance).Instance(key),
				ActionTriggerEvent:      *triggeringEvent,
				ActionTriggerBlockIndex: n.actionTriggerBlockIndex,
				ActionsListIndex:        n.actionListIndex,
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
		Subject: n.resourceAddress.Resource,
	})

	refs = append(refs, &addrs.Reference{
		Subject: n.actionAddress.Action,
	})

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

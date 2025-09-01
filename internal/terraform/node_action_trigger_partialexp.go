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

// NodeActionTriggerPartialExpanded is a graph node that stands in for
// an unbounded set of potential action trigger instances that we don't yet know.
//
// Its job is to check the configuration as much as we can with the information
// that's available (so we can raise an error early if something is clearly
// wrong across _all_ potential instances) and to record a placeholder value
// for use when evaluating other objects that refer to this resource.
//
// This is the partial-expanded equivalent of NodeActionTriggerInstance.
type NodeActionTriggerPartialExpanded struct {
	addr                   addrs.PartialExpandedAction
	config                 *configs.Action
	resolvedProvider       addrs.AbsProviderConfig
	lifecycleActionTrigger *lifecycleActionTriggerPartialExpanded
}

type lifecycleActionTriggerPartialExpanded struct {
	resourceAddress         addrs.PartialExpandedResource
	events                  []configs.ActionTriggerEvent
	actionTriggerBlockIndex int
	actionListIndex         int
	invokingSubject         *hcl.Range
}

func (at *lifecycleActionTriggerPartialExpanded) Name() string {
	return fmt.Sprintf("%s.lifecycle.action_trigger[%d].actions[%d]", at.resourceAddress.String(), at.actionTriggerBlockIndex, at.actionListIndex)
}

var (
	_ graphNodeEvalContextScope = (*NodeActionTriggerPartialExpanded)(nil)
	_ GraphNodeExecutable       = (*NodeActionTriggerPartialExpanded)(nil)
)

// Name implements [dag.NamedVertex].
func (n *NodeActionTriggerPartialExpanded) Name() string {
	return n.addr.String()
}

// Path implements graphNodeEvalContextScope.
func (n *NodeActionTriggerPartialExpanded) Path() evalContextScope {
	if moduleAddr, ok := n.addr.ModuleInstance(); ok {
		return evalContextModuleInstance{Addr: moduleAddr}
	} else if moduleAddr, ok := n.addr.PartialExpandedModule(); ok {
		return evalContextPartialExpandedModule{Addr: moduleAddr}
	} else {
		// Should not get here: at least one of the two cases above
		// should always be true for any valid addrs.PartialExpandedResource
		panic("addrs.PartialExpandedResource has neither a partial-expanded or a fully-expanded module instance address")
	}
}

func (n *NodeActionTriggerPartialExpanded) ActionAddr() addrs.ConfigAction {
	return n.addr.ConfigAction()
}

// Execute implements GraphNodeExecutable.
func (n *NodeActionTriggerPartialExpanded) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	// We know that if the action is partially expanded, the triggering resource must also be partially expanded.
	partialResourceChange := ctx.Deferrals().GetDeferredPartialExpandedResource(n.lifecycleActionTrigger.resourceAddress)
	if partialResourceChange == nil {
		panic("partialResource is nil")
	}

	triggeringEvent, isTriggered := actionIsTriggeredByEvent(n.lifecycleActionTrigger.events, partialResourceChange.Change.Action)
	if !isTriggered {
		return nil
	}

	actionInstance, ok := ctx.Actions().GetPartialExpandedAction(n.addr)
	if !ok {
		panic("action is nil")
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

	// We remove the marks for planning, we will record the sensitive values in the plans.ActionInvocationInstance
	unmarkedConfig, _ := actionInstance.ConfigValue.UnmarkDeepWithPaths()

	resp := provider.PlanAction(providers.PlanActionRequest{
		ActionType:         n.addr.ConfigAction().Action.Type,
		ProposedActionData: unmarkedConfig,
		ClientCapabilities: ctx.ClientCapabilities(),
	})

	if resp.Diagnostics.HasErrors() {
		diags = diags.Append(resp.Diagnostics)
		return diags
	}

	ctx.Deferrals().ReportPartialActionInvocationDeferred(plans.PartialExpandedActionInvocationInstance{
		Addr:         n.addr,
		ProviderAddr: n.resolvedProvider,
		ActionTrigger: plans.PartialLifecycleActionTrigger{
			TriggeringResourceAddr:  n.lifecycleActionTrigger.resourceAddress,
			ActionTriggerEvent:      *triggeringEvent,
			ActionTriggerBlockIndex: n.lifecycleActionTrigger.actionTriggerBlockIndex,
			ActionsListIndex:        n.lifecycleActionTrigger.actionListIndex,
		},
		ConfigValue: actionInstance.ConfigValue,
	}, providers.DeferredReasonInstanceCountUnknown)
	return nil
}

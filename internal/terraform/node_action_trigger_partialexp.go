// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/ephemeral"
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

	triggeringEvents := actionIsTriggeredByEvent(n.lifecycleActionTrigger.events, partialResourceChange.Change.Action)
	if len(triggeringEvents) == 0 {
		return nil
	}

	provider, schema, err := getProvider(ctx, n.resolvedProvider)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Failed to get provider for %s", n.addr),
			Detail:   fmt.Sprintf("Failed to get provider: %s", err),
			Subject:  &n.config.DeclRange,
		})
		return diags
	}

	actionSchema, ok := schema.Actions[n.addr.ConfigAction().Action.Type]
	if !ok {
		// This should have been caught earlier
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Action %s not found in provider schema", n.addr),
			Detail:   fmt.Sprintf("The action %s was not found in the provider schema for %s", n.addr.ConfigAction().Action.Type, n.resolvedProvider),
			Subject:  &n.config.DeclRange,
		})
		return diags
	}

	expander := ctx.InstanceExpander()
	keyData := expander.GetActionInstanceRepetitionData(n.addr.UnknownActionInstance())
	configVal := cty.NullVal(actionSchema.ConfigSchema.ImpliedType())
	if n.config.Config != nil {
		var configDiags tfdiags.Diagnostics
		configVal, _, configDiags = ctx.EvaluateBlock(n.config.Config, actionSchema.ConfigSchema, nil, keyData)

		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			return diags
		}

		valDiags := validateResourceForbiddenEphemeralValues(ctx, configVal, actionSchema.ConfigSchema)
		diags = diags.Append(valDiags.InConfigBody(n.config.Config, n.addr.String()))

		if valDiags.HasErrors() {
			return diags
		}

		_, deprecationDiags := ctx.Deprecations().ValidateAndUnmarkConfig(configVal, actionSchema.ConfigSchema, n.ActionAddr().Module)
		diags = diags.Append(deprecationDiags.InConfigBody(n.config.Config, n.addr.String()))
	}

	// We remove the marks for planning, we will record the sensitive values in the plans.ActionInvocationInstance
	unmarkedConfig, _ := configVal.UnmarkDeepWithPaths()

	resp := provider.PlanAction(providers.PlanActionRequest{
		ActionType:         n.addr.ConfigAction().Action.Type,
		ProposedActionData: unmarkedConfig,
		ClientCapabilities: ctx.ClientCapabilities(),
	})

	if resp.Diagnostics.HasErrors() {
		diags = diags.Append(resp.Diagnostics)
		return diags
	}

	for _, triggeringEvent := range triggeringEvents {
		ctx.Deferrals().ReportActionInvocationDeferred(plans.ActionInvocationInstance{
			Addr:         n.addr.UnknownActionInstance(),
			ProviderAddr: n.resolvedProvider,
			ActionTrigger: &plans.ResourceActionTrigger{
				TriggeringResourceAddr:  n.lifecycleActionTrigger.resourceAddress.UnknownResourceInstance(),
				ActionTriggerEvent:      triggeringEvent,
				ActionTriggerBlockIndex: n.lifecycleActionTrigger.actionTriggerBlockIndex,
				ActionsListIndex:        n.lifecycleActionTrigger.actionListIndex,
			},
			ConfigValue: ephemeral.RemoveEphemeralValues(configVal),
		}, providers.DeferredReasonInstanceCountUnknown)
	}
	return nil
}

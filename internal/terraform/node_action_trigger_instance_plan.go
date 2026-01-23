// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/ephemeral"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	_ GraphNodeExecutable = (*nodeActionTriggerPlanInstance)(nil)
	_ GraphNodeModulePath = (*nodeActionTriggerPlanInstance)(nil)
)

type nodeActionTriggerPlanInstance struct {
	actionAddress    addrs.AbsActionInstance
	resolvedProvider addrs.AbsProviderConfig
	actionConfig     *configs.Action

	actionTriggerConfig lifecycleActionTriggerConfig
}

type lifecycleActionTriggerConfig struct {
	resourceAddress         addrs.AbsResourceInstance
	events                  []configs.ActionTriggerEvent
	actionTriggerBlockIndex int
	actionListIndex         int
	invokingSubject         *hcl.Range
	conditionExpr           hcl.Expression
}

func (at *lifecycleActionTriggerConfig) Name() string {
	return fmt.Sprintf("%s.lifecycle.action_trigger[%d].actions[%d]", at.resourceAddress.String(), at.actionTriggerBlockIndex, at.actionListIndex)
}

func (at *lifecycleActionTriggerConfig) ActionTrigger(triggeringEvent configs.ActionTriggerEvent) *plans.LifecycleActionTrigger {
	return &plans.LifecycleActionTrigger{
		TriggeringResourceAddr:  at.resourceAddress,
		ActionTriggerBlockIndex: at.actionTriggerBlockIndex,
		ActionsListIndex:        at.actionListIndex,
		ActionTriggerEvent:      triggeringEvent,
	}
}

var (
	_ GraphNodeModuleInstance = (*nodeActionTriggerPlanInstance)(nil)
	_ GraphNodeExecutable     = (*nodeActionTriggerPlanInstance)(nil)
)

func (n *nodeActionTriggerPlanInstance) Name() string {
	return fmt.Sprintf("%s triggered by %s", n.actionAddress.String(), n.actionTriggerConfig.resourceAddress.String())
}

func (n *nodeActionTriggerPlanInstance) Execute(ctx EvalContext, operation walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	deferrals := ctx.Deferrals()

	// We need the action invocation early to check if we need to
	ai := plans.ActionInvocationInstance{
		Addr:          n.actionAddress,
		ActionTrigger: n.actionTriggerConfig.ActionTrigger(configs.Unknown),
	}
	change := ctx.Changes().GetResourceInstanceChange(n.actionTriggerConfig.resourceAddress, n.actionTriggerConfig.resourceAddress.CurrentObject().DeposedKey)

	deferred, moreDiags := deferrals.ShouldDeferActionInvocation(ai, n.actionTriggerConfig.invokingSubject)
	diags = diags.Append(moreDiags)
	if deferred {
		deferrals.ReportActionInvocationDeferred(ai, providers.DeferredReasonDeferredPrereq)
		return diags
	}

	if moreDiags.HasErrors() {
		return diags
	}

	if change == nil {
		// nothing to do (this may be a refresh )
		return diags
	}

	actionInstance, ok := ctx.Actions().GetActionInstance(n.actionAddress)
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to non-existent action instance",
			Detail:   "Action instance was not found in the current context.",
			Subject:  n.actionTriggerConfig.invokingSubject,
		})
		return diags
	}

	ai.ProviderAddr = actionInstance.ProviderAddr
	// with resources, the provider would be expected to strip the ephemeral
	// values out. with actions, we don't get the value back from the
	// provider so we'll do that ourselves now.
	ai.ConfigValue = ephemeral.RemoveEphemeralValues(actionInstance.ConfigValue)

	triggeredEvents := actionIsTriggeredByEvent(n.actionTriggerConfig.events, change.Action)
	if len(triggeredEvents) == 0 {
		return diags
	}

	// Evaluate the condition expression if it exists (otherwise it's true)
	if n.actionTriggerConfig.conditionExpr != nil {
		condition, conditionDiags := evaluateActionCondition(ctx, actionConditionContext{
			events:          n.actionTriggerConfig.events,
			conditionExpr:   n.actionTriggerConfig.conditionExpr,
			resourceAddress: n.actionTriggerConfig.resourceAddress,
		})
		diags = diags.Append(conditionDiags)
		if conditionDiags.HasErrors() {
			return conditionDiags
		}

		// The condition is false so we skip the action
		if !condition {
			return diags
		}
	}

	provider, _, err := getProvider(ctx, actionInstance.ProviderAddr)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to get provider",
			Detail:   fmt.Sprintf("Failed to get provider: %s", err),
			Subject:  n.actionTriggerConfig.invokingSubject,
		})

		return diags
	}

	// We remove the marks for planning, we will record the sensitive values in the plans.ActionInvocationInstance
	unmarkedConfig, _ := actionInstance.ConfigValue.UnmarkDeepWithPaths()

	cc := ctx.ClientCapabilities()
	cc.DeferralAllowed = false // for now, deferrals in actions are always disabled
	resp := provider.PlanAction(providers.PlanActionRequest{
		ActionType:         n.actionAddress.Action.Action.Type,
		ProposedActionData: unmarkedConfig,
		ClientCapabilities: cc,
	})

	if len(resp.Diagnostics) > 0 {
		severity := hcl.DiagWarning
		message := "Warnings when planning action"
		err := resp.Diagnostics.Warnings().ErrWithWarnings()
		if resp.Diagnostics.HasErrors() {
			severity = hcl.DiagError
			message = "Failed to plan action"
			err = resp.Diagnostics.ErrWithWarnings()
		}

		diags = diags.Append(&hcl.Diagnostic{
			Severity: severity,
			Summary:  message,
			Detail:   err.Error(),
			Subject:  n.actionTriggerConfig.invokingSubject,
		})
	}
	if resp.Deferred != nil {
		// we always set allow_deferrals to be false for actions, so this
		// should not happen
		diags = diags.Append(deferring.UnexpectedProviderDeferralDiagnostic(n.actionAddress))
	}
	if resp.Diagnostics.HasErrors() {
		return diags
	}

	// We are planning to run this action multiple times so
	for _, triggeredEvent := range triggeredEvents {
		eventSpecificAi := ai.DeepCopy()
		// We need to set the triggering event on the action invocation
		eventSpecificAi.ActionTrigger = n.actionTriggerConfig.ActionTrigger(triggeredEvent)
		ctx.Changes().AppendActionInvocation(eventSpecificAi)
	}
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

type actionConditionContext struct {
	events          []configs.ActionTriggerEvent
	conditionExpr   hcl.Expression
	resourceAddress addrs.AbsResourceInstance
}

func evaluateActionCondition(ctx EvalContext, at actionConditionContext) (bool, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	rd := instances.RepetitionData{}
	refs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRef, at.conditionExpr)
	diags = diags.Append(refDiags)
	if diags.HasErrors() {
		return false, diags
	}

	for _, ref := range refs {
		if ref.Subject == addrs.Self {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Self reference not allowed",
				Detail:   `The condition expression cannot reference "self".`,
				Subject:  at.conditionExpr.Range().Ptr(),
			})
		}
	}

	if diags.HasErrors() {
		return false, diags
	}

	if containsBeforeEvent(at.events) {
		// If events contains a before event we want to error if count or each is used
		for _, ref := range refs {
			if _, ok := ref.Subject.(addrs.CountAttr); ok {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Count reference not allowed",
					Detail:   `The condition expression cannot reference "count" if the action is run before the resource is applied.`,
					Subject:  at.conditionExpr.Range().Ptr(),
				})
			}

			if _, ok := ref.Subject.(addrs.ForEachAttr); ok {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Each reference not allowed",
					Detail:   `The condition expression cannot reference "each" if the action is run before the resource is applied.`,
					Subject:  at.conditionExpr.Range().Ptr(),
				})
			}

			if diags.HasErrors() {
				return false, diags
			}
		}
	} else {
		// If there are only after events we allow self, count, and each
		expander := ctx.InstanceExpander()
		rd = expander.GetResourceInstanceRepetitionData(at.resourceAddress)
	}

	scope := ctx.EvaluationScope(nil, nil, rd)
	val, conditionEvalDiags := scope.EvalExpr(at.conditionExpr, cty.Bool)
	diags = diags.Append(conditionEvalDiags)
	if diags.HasErrors() {
		return false, diags
	}

	if !val.IsWhollyKnown() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Condition must be known",
			Detail:   "The condition expression resulted in an unknown value, but it must be a known boolean value.",
			Subject:  at.conditionExpr.Range().Ptr(),
		})
		return false, diags
	}

	return val.True(), nil
}

func containsBeforeEvent(events []configs.ActionTriggerEvent) bool {
	for _, event := range events {
		switch event {
		case configs.BeforeCreate, configs.BeforeUpdate:
			return true
		default:
			continue
		}
	}
	return false
}

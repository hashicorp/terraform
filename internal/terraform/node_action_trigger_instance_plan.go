// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
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
	conditionExpr           hcl.Expression
}

func (at *lifecycleActionTriggerInstance) Name() string {
	return fmt.Sprintf("%s.lifecycle.action_trigger[%d].actions[%d]", at.resourceAddress.String(), at.actionTriggerBlockIndex, at.actionListIndex)
}

func (at *lifecycleActionTriggerInstance) ActionTrigger(triggeringEvent configs.ActionTriggerEvent) plans.LifecycleActionTrigger {
	return plans.LifecycleActionTrigger{
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
	deferrals := ctx.Deferrals()

	if n.lifecycleActionTrigger == nil {
		panic("Only actions triggered by plan and apply are supported")
	}

	actionInstance, ok := ctx.Actions().GetActionInstance(n.actionAddress)
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to non-existent action instance",
			Detail:   "Action instance was not found in the current context.",
			Subject:  n.lifecycleActionTrigger.invokingSubject,
		})
		return diags
	}

	// We need the action invocation early to check if we need to
	ai := plans.ActionInvocationInstance{
		Addr:          n.actionAddress,
		ProviderAddr:  actionInstance.ProviderAddr,
		ActionTrigger: n.lifecycleActionTrigger.ActionTrigger(configs.Unknown),
		ConfigValue:   actionInstance.ConfigValue,
	}

	// If we already deferred an action invocation on the same resource with an earlier trigger we can defer this one as well
	if deferrals.DeferralAllowed() && deferrals.ShouldDeferActionInvocation(ai) {
		deferrals.ReportActionInvocationDeferred(ai, providers.DeferredReasonDeferredPrereq)
		return diags
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

	// Evaluate the condition expression if it exists (otherwise it's true)
	if n.lifecycleActionTrigger != nil && n.lifecycleActionTrigger.conditionExpr != nil {
		condition, conditionDiags := evaluateCondition(ctx, n.lifecycleActionTrigger.conditionExpr)
		diags = diags.Append(conditionDiags)
		if conditionDiags.HasErrors() {
			return conditionDiags
		}

		// The condition is false so we skip the action
		if condition.False() {
			return diags
		}
	}

	// We need to set the triggering event on the action invocation
	ai.ActionTrigger = n.lifecycleActionTrigger.ActionTrigger(*triggeringEvent)

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
	unmarkedConfig, pvms := actionInstance.ConfigValue.UnmarkDeepWithPaths()
	// We only support sensitive marks, all other marks cause an error
	_, otherMarks := marks.PathsWithMark(pvms, marks.Sensitive)
	if len(otherMarks) > 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported marks",
			Detail:   "Only sensitive marks are supported in action configuration",
			Subject:  &n.actionConfig.DeclRange,
		})
		return diags
	}

	resp := provider.PlanAction(providers.PlanActionRequest{
		ActionType:         n.actionAddress.Action.Action.Type,
		ProposedActionData: unmarkedConfig,
		ClientCapabilities: ctx.ClientCapabilities(),
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
			Subject:  n.lifecycleActionTrigger.invokingSubject,
		})
	}
	if resp.Deferred != nil && !deferrals.DeferralAllowed() {
		diags = diags.Append(deferring.UnexpectedProviderDeferralDiagnostic(n.actionAddress))
	}
	if resp.Diagnostics.HasErrors() {
		return diags
	}

	if resp.Deferred != nil {
		deferrals.ReportActionInvocationDeferred(ai, resp.Deferred.Reason)

		// If we run as part of a before action we need to retrospectively defer the triggering resource
		// For this we remove the change and report the deferral
		ctx.Changes().RemoveResourceInstanceChange(change.Addr, change.Addr.CurrentObject().DeposedKey)
		deferrals.ReportResourceInstanceDeferred(change.Addr, providers.DeferredReasonDeferredPrereq, change)
		return diags
	}

	ctx.Changes().AppendActionInvocation(&ai)
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

func evaluateCondition(ctx EvalContext, conditionExpr hcl.Expression) (cty.Value, tfdiags.Diagnostics) {
	// TODO: Support self in conditions
	val, diags := ctx.EvaluateExpr(conditionExpr, cty.Bool, nil)
	if diags.HasErrors() {
		return cty.False, diags
	}

	// TODO: Support unknown condition values
	if !val.IsWhollyKnown() {
		panic("condition is not wholly known")
	}
	// If the condition is neither true nor false, it's an error
	if !(val.True() || val.False()) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid condition",
			Detail:   "The condition must be either true or false",
			Subject:  conditionExpr.Range().Ptr(),
		})
		return cty.False, diags
	}

	return val, nil
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/ephemeral"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeActionTriggerApply struct {
	ActionInvocation   *plans.ActionInvocationInstanceSrc
	resolvedProvider   addrs.AbsProviderConfig
	ActionTriggerRange *hcl.Range
	ConditionExpr      hcl.Expression
}

var (
	_ GraphNodeExecutable = (*nodeActionTriggerApply)(nil)
	_ GraphNodeReferencer = (*nodeActionTriggerApply)(nil)
)

func (n *nodeActionTriggerApply) Name() string {
	return n.ActionInvocation.Addr.String() + " (instance)"
}

func (n *nodeActionTriggerApply) Execute(ctx EvalContext, wo walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	actionInvocation := n.ActionInvocation

	if n.ConditionExpr != nil {
		// We know this must be a lifecycle action, otherwise we would have no condition
		at := actionInvocation.ActionTrigger.(*plans.LifecycleActionTrigger)
		condition, conditionDiags := evaluateActionCondition(ctx, actionConditionContext{
			// For applying the triggering event is sufficient, if the condition could not have
			// been evaluated due to in invalid mix of events we would have caught it durin planning.
			events:          []configs.ActionTriggerEvent{at.ActionTriggerEvent},
			conditionExpr:   n.ConditionExpr,
			resourceAddress: at.TriggeringResourceAddr,
		})
		diags = diags.Append(conditionDiags)
		if diags.HasErrors() {
			return diags
		}

		if !condition {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Condition changed evaluation during apply",
				Detail:   "The condition evaluated to false during apply, but was true during planning. This may lead to unexpected behavior.",
				Subject:  n.ConditionExpr.Range().Ptr(),
			})
		}
	}

	ai := ctx.Changes().GetActionInvocation(actionInvocation.Addr, actionInvocation.ActionTrigger)
	if ai == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Action invocation not found in plan",
			Detail:   "Could not find action invocation for address " + actionInvocation.Addr.String(),
			Subject:  n.ActionTriggerRange,
		})
		return diags
	}
	actionData, ok := ctx.Actions().GetActionInstance(ai.Addr)
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Action instance not found",
			Detail:   "Could not find action instance for address " + ai.Addr.String(),
			Subject:  n.ActionTriggerRange,
		})
		return diags
	}
	provider, schema, err := getProvider(ctx, actionData.ProviderAddr)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Failed to get provider for %s", ai.Addr),
			Detail:   fmt.Sprintf("Failed to get provider: %s", err),
			Subject:  n.ActionTriggerRange,
		})
		return diags
	}

	actionSchema, ok := schema.Actions[ai.Addr.Action.Action.Type]
	if !ok {
		// This should have been caught earlier
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Action %s not found in provider schema", ai.Addr),
			Detail:   fmt.Sprintf("The action %s was not found in the provider schema for %s", ai.Addr.Action.Action.Type, actionData.ProviderAddr),
			Subject:  n.ActionTriggerRange,
		})
		return diags
	}

	configValue := actionData.ConfigValue

	// Validate that what we planned matches the action data we have.
	errs := objchange.AssertObjectCompatible(actionSchema.ConfigSchema, ai.ConfigValue, ephemeral.RemoveEphemeralValues(configValue))
	for _, err := range errs {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider produced inconsistent final plan",
			Detail: fmt.Sprintf("When expanding the plan for %s to include new values learned so far during apply, Terraform produced an invalid new value for %s.\n\nThis is a bug in Terraform, which should be reported.",
				ai.Addr, tfdiags.FormatError(err)),
			Subject: n.ActionTriggerRange,
		})
	}

	hookIdentity := HookActionIdentity{
		Addr:          ai.Addr,
		ActionTrigger: ai.ActionTrigger,
	}

	ctx.Hook(func(h Hook) (HookAction, error) {
		return h.StartAction(hookIdentity)
	})

	// We don't want to send the marks, but all marks are okay in the context
	// of an action invocation. We can't reuse our ephemeral free value from
	// above because we want the ephemeral values to be included.
	unmarkedConfigValue, _ := configValue.UnmarkDeep()
	resp := provider.InvokeAction(providers.InvokeActionRequest{
		ActionType:         ai.Addr.Action.Action.Type,
		PlannedActionData:  unmarkedConfigValue,
		ClientCapabilities: ctx.ClientCapabilities(),
	})

	respDiags := n.AddSubjectToDiagnostics(resp.Diagnostics)
	diags = diags.Append(respDiags)
	if respDiags.HasErrors() {
		ctx.Hook(func(h Hook) (HookAction, error) {
			return h.CompleteAction(hookIdentity, respDiags.Err())
		})
		return diags
	}

	for event := range resp.Events {
		switch ev := event.(type) {
		case providers.InvokeActionEvent_Progress:
			ctx.Hook(func(h Hook) (HookAction, error) {
				return h.ProgressAction(hookIdentity, ev.Message)
			})
		case providers.InvokeActionEvent_Completed:
			// Enhance the diagnostics
			diags = diags.Append(n.AddSubjectToDiagnostics(ev.Diagnostics))
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
		Subject: n.ActionInvocation.Addr.Action,
	})

	conditionRefs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRef, n.ConditionExpr)
	if refDiags.HasErrors() {
		panic(fmt.Sprintf("error parsing references in expression: %v", refDiags))
	}
	if conditionRefs != nil {
		refs = append(refs, conditionRefs...)
	}

	return refs
}

// GraphNodeModulePath
func (n *nodeActionTriggerApply) ModulePath() addrs.Module {
	return n.ActionInvocation.Addr.Module.Module()
}

// GraphNodeModuleInstance
func (n *nodeActionTriggerApply) Path() addrs.ModuleInstance {
	return n.ActionInvocation.Addr.Module
}

func (n *nodeActionTriggerApply) AddSubjectToDiagnostics(input tfdiags.Diagnostics) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if len(input) > 0 {
		severity := hcl.DiagWarning
		message := "Warning when invoking action"
		err := input.Warnings().ErrWithWarnings()
		if input.HasErrors() {
			severity = hcl.DiagError
			message = "Error when invoking action"
			err = input.ErrWithWarnings()
		}

		diags = diags.Append(&hcl.Diagnostic{
			Severity: severity,
			Summary:  message,
			Detail:   err.Error(),
			Subject:  n.ActionTriggerRange,
		})
	}
	return diags
}

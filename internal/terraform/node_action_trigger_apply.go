// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
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
	return "action_apply_" + n.ActionInvocation.Addr.String()
}

func (n *nodeActionTriggerApply) Execute(ctx EvalContext, wo walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	actionInvocation := n.ActionInvocation

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

	var self *addrs.ResourceInstance
	if at, ok := n.ActionInvocation.ActionTrigger.(plans.LifecycleActionTrigger); ok {
		self = &at.TriggeringResourceAddr.Resource
	}

	if n.ConditionExpr != nil {
		scope := ctx.EvaluationScope(self, nil, ai.ConditionRepetitionData)
		condition, conditionDiags := scope.EvalExpr(n.ConditionExpr, cty.Bool)
		diags = diags.Append(conditionDiags)
		if diags.HasErrors() {
			return diags
		}
		if !condition.IsWhollyKnown() {
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Condition expression is not known",
				Detail:   "During apply the condition expression must be known, and must evaluate to a boolean value",
				Subject:  n.ConditionExpr.Range().Ptr(),
			})
		}
		// If the condition evaluates to false, skip the action
		if condition.False() {
			return diags
		}
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

	// We don't want to send the marks, but all marks are okay in the context of an action invocation.
	unmarkedConfigValue, _ := actionData.ConfigValue.UnmarkDeep()

	// Validate that what we planned matches the action data we have.
	errs := objchange.AssertObjectCompatible(actionSchema.ConfigSchema, ai.ConfigValue, unmarkedConfigValue)
	for _, err := range errs {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider produced inconsistent final plan",
			Detail: fmt.Sprintf("When expanding the plan for %s to include new values learned so far during apply, provider %q produced an invalid new value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				ai.Addr, actionData.ProviderAddr.Provider.String(), tfdiags.FormatError(err)),
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

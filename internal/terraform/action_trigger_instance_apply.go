// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type actionTriggerApplyInstance struct {
	ActionInvocation *plans.ActionInvocationInstanceSrc
	resolvedProvider addrs.AbsProviderConfig

	// actionNode links the trigger to it's action config node.
	// This is connected by the diff transformer.
	actionNode *NodeActionConfig
}

var (
	// this doesn't operate as an independent node in the graph, but we obtain
	// the relevant information for evaluataion via these interfaces
	_ GraphNodeReferencer       = (*actionTriggerApplyInstance)(nil)
	_ GraphNodeProviderConsumer = (*actionTriggerApplyInstance)(nil)
)

func (n *actionTriggerApplyInstance) invoke(ctx EvalContext, wo walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	provider, _, err := getProvider(ctx, n.resolvedProvider)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Failed to get provider for %s", n.resolvedProvider),
			Detail:   fmt.Sprintf("Failed to get provider: %s", err),
			Subject:  n.actionNode.Config.DeclRange.Ptr(),
		})
		return diags
	}

	if n.actionNode == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Invoke %s missing action config", n.ActionInvocation.Addr),
			Detail:   fmt.Sprintf("The action config was not found for invocation %s", n.ActionInvocation.Addr),
			Subject:  n.actionNode.Config.DeclRange.Ptr(),
		})
		return diags
	}

	// FIXME: missing action trigger reference for diags
	configValue, actionDiags := n.actionNode.EvalInstance(ctx, n.ActionInvocation.Addr.Action.Key, nil)
	diags = diags.Append(actionDiags)
	if diags.HasErrors() {
		return diags
	}

	if !configValue.IsWhollyKnown() {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Action configuration unknown during apply",
			Detail:   fmt.Sprintf("The action %s was not fully known during apply.\n\nThis is a bug in Terraform, please report it.", n.ActionInvocation.Addr),
			// FIXME: maybe turn this into an attribute path diagnostic?
			Subject: n.actionNode.Config.DeclRange.Ptr(),
		})
	}

	hookIdentity := HookActionIdentity{
		Addr:          n.ActionInvocation.Addr,
		ActionTrigger: n.ActionInvocation.ActionTrigger,
	}

	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.StartAction(hookIdentity)
	}))
	if diags.HasErrors() {
		return diags
	}

	// We don't want to send the marks, but all marks are okay in the context
	// of an action invocation. We can't reuse our ephemeral free value from
	// above because we want the ephemeral values to be included.
	unmarkedConfigValue, _ := configValue.UnmarkDeep()
	resp := provider.InvokeAction(providers.InvokeActionRequest{
		ActionType:         n.ActionInvocation.Addr.Action.Action.Type,
		PlannedActionData:  unmarkedConfigValue,
		ClientCapabilities: ctx.ClientCapabilities(),
	})

	respDiags := n.AddSubjectToDiagnostics(resp.Diagnostics)
	diags = diags.Append(respDiags)
	if respDiags.HasErrors() {
		diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
			return h.CompleteAction(hookIdentity, respDiags.Err())
		}))
		return diags
	}

	if resp.Events != nil { // should only occur in misconfigured tests
		for event := range resp.Events {
			switch ev := event.(type) {
			case providers.InvokeActionEvent_Progress:
				diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
					return h.ProgressAction(hookIdentity, ev.Message)
				}))
				if diags.HasErrors() {
					return diags
				}
			case providers.InvokeActionEvent_Completed:
				// Enhance the diagnostics
				diags = diags.Append(n.AddSubjectToDiagnostics(ev.Diagnostics))
				diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
					return h.CompleteAction(hookIdentity, ev.Diagnostics.Err())
				}))
				if ev.Diagnostics.HasErrors() {
					return diags
				}
				if diags.HasErrors() {
					return diags
				}
			default:
				panic(fmt.Sprintf("unexpected action event type %T", ev))
			}
		}
	} else {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider return invalid response",
			Detail:   "Provider response did not include any events",
			Subject:  n.actionNode.Config.DeclRange.Ptr(),
		})
	}

	return diags
}

func (n *actionTriggerApplyInstance) Provider() ProviderRef {
	return ProviderRef{
		Addr:     n.ActionInvocation.ProviderAddr,
		Resolved: true,
	}
}

func (n *actionTriggerApplyInstance) SetProvider(config addrs.AbsProviderConfig) {
	n.resolvedProvider = config
}

func (n *actionTriggerApplyInstance) References() []*addrs.Reference {
	var refs []*addrs.Reference

	refs = append(refs, &addrs.Reference{
		Subject: n.ActionInvocation.Addr.Action,
	})

	refs = append(refs, n.actionNode.References()...)

	return refs
}

// GraphNodeReferencer
func (n *actionTriggerApplyInstance) ModulePath() addrs.Module {
	return n.ActionInvocation.Addr.Module.Module()
}

// GraphNodeExecutable
func (n *actionTriggerApplyInstance) Path() addrs.ModuleInstance {
	return n.ActionInvocation.Addr.Module
}

func (n *actionTriggerApplyInstance) AddSubjectToDiagnostics(input tfdiags.Diagnostics) tfdiags.Diagnostics {
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

			// FIXME: this is the action config block, make sure user can associate this with the trigger
			Subject: n.actionNode.Config.DeclRange.Ptr(),
		})
	}
	return diags
}

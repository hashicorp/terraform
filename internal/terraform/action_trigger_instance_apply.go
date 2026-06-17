// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type actionTriggerApplyInstance struct {
	ActionInvocation *plans.ActionInvocationInstanceSrc

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

func (n *actionTriggerApplyInstance) Invoke(ctx EvalContext, caller addrs.Referenceable, callerVal cty.Value, fromPlan bool) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	diagsWith := n.ActionInvocation.Addr.String()
	if caller != nil {
		diagsWith = fmt.Sprintf("%s, called from %s", diagsWith, caller)
	}

	provider, actionProviderSchema, err := getProvider(ctx, n.ActionInvocation.ProviderAddr)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Failed to get provider for %s", n.ActionInvocation.ProviderAddr),
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

	var configVal cty.Value

	// fromPlan indicates we can use the entire planned value for the action,
	// and should not attempt to reevaluate the config.
	if fromPlan {
		actionSchema, ok := actionProviderSchema.Actions[n.ActionInvocation.Addr.Action.Action.Type]
		if !ok {
			// This should have been caught earlier, but we don't want to panic
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Action %s not found in provider schema", n.ActionInvocation.Addr),
				Detail:   fmt.Sprintf("The action %s was not found in the provider schema for %s", n.ActionInvocation.Addr, n.ActionInvocation.ProviderAddr),
				Subject:  n.actionNode.Config.DeclRange.Ptr(),
			})
			return diags
		}
		inv, err := n.ActionInvocation.Decode(&actionSchema)
		if err != nil {
			return diags.Append(err)
		}
		configVal = inv.ConfigValue
	} else {
		val, actionDiags := n.actionNode.EvalInstance(ctx, n.ActionInvocation.Addr, nil, caller, callerVal)
		diags = diags.Append(actionDiags)
		if diags.HasErrors() {
			return diags
		}
		configVal = val
	}

	if !configVal.IsWhollyKnown() {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Action configuration unknown during apply",
			Detail: fmt.Sprintf("The action %s was not fully known during apply. "+
				"This may be caused by using the caller object in conjunction with a before event.", n.ActionInvocation.Addr),
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
	unmarkedConfigValue, _ := configVal.UnmarkDeep()
	resp := provider.InvokeAction(providers.InvokeActionRequest{
		ActionType:         n.ActionInvocation.Addr.Action.Action.Type,
		PlannedActionData:  unmarkedConfigValue,
		ClientCapabilities: ctx.ClientCapabilities(),
	})

	diags = diags.Append(resp.Diagnostics.InConfigBody(n.actionNode.Config.Body, diagsWith))
	if diags.HasErrors() {
		diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
			return h.CompleteAction(hookIdentity, resp.Diagnostics.Err())
		}))
		return diags
	}

	if resp.Events != nil {
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
				diags = diags.Append(ev.Diagnostics.InConfigBody(n.actionNode.Config.Body, diagsWith))

				diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
					return h.CompleteAction(hookIdentity, ev.Diagnostics.Err())
				}))
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
	// keep this method to satisfy GraphNodeProviderConsumer, but we already
	// have a resolved provider saved in the plan
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

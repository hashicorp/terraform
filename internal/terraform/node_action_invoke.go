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

var (
	_ GraphNodeDynamicExpandable      = (*nodeActionInvokeExpand)(nil)
	_ GraphNodeReferencer             = (*nodeActionInvokeExpand)(nil)
	_ GraphNodeActionProviderConsumer = (*nodeActionInvokeExpand)(nil)
)

type nodeActionInvokeExpand struct {
	// invoke always relies on targeting, and we need to capture the initial
	// target here to ensure we only expand the targeted instances
	Target addrs.Targetable

	Module addrs.Module

	// as we have used in other targeting situations, because a single instance
	// is indistinguishable from an expanded block, we'll default to instance
	// addrs for consistency.
	Addr         addrs.AbsActionInstance
	ActionConfig *NodeActionConfig
}

func (n *nodeActionInvokeExpand) ActionProviders() []ProviderRef {
	return []ProviderRef{ProviderRef{
		Addr:     n.ActionConfig.ResolvedProvider,
		Resolved: true,
	}}
}

func (n *nodeActionInvokeExpand) ModulePath() addrs.Module {
	return n.Module
}

func (n *nodeActionInvokeExpand) Name() string {
	module := n.ModulePath().String()
	if len(module) > 0 {
		module = module + "."
	}
	return fmt.Sprintf("%sinvoke.%s", module, n.Addr)
}

func (n *nodeActionInvokeExpand) References() []*addrs.Reference {
	return []*addrs.Reference{
		{
			Subject: n.Addr.Action,
		},
		{
			Subject: n.Addr.Action.Action,
		},
	}
}

func (n *nodeActionInvokeExpand) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var g Graph

	// the nodeActionInvokeExpand is only here to expand within any targeted
	// modules, becuase the action expansion and evaluation must happen within
	// that module instance's scope
	expander := ctx.InstanceExpander()

	for _, mod := range expander.ExpandModule(n.Module, false) {
		if !mod.TargetContains(n.Target) {
			continue
		}

		g.Add(&nodeActionPlanInvoke{
			Module:       mod,
			Addr:         n.Addr,
			ActionConfig: n.ActionConfig,
			ProviderAddr: n.ActionConfig.ResolvedProvider,
		})
	}
	addRootNodeToGraph(&g)

	return &g, nil
}

var (
	_ GraphNodeExecutable     = (*nodeActionPlanInvoke)(nil)
	_ GraphNodeModuleInstance = (*nodeActionPlanInvoke)(nil)
)

type nodeActionPlanInvoke struct {
	Module       addrs.ModuleInstance
	Addr         addrs.AbsActionInstance
	ActionConfig *NodeActionConfig
	ProviderAddr addrs.AbsProviderConfig
}

func (n *nodeActionPlanInvoke) Path() addrs.ModuleInstance {
	return n.Module
}

func (n *nodeActionPlanInvoke) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
	// for now each action instance will be invoked serially
	return n.invokeActions(ctx)
}

func (n *nodeActionPlanInvoke) invokeActions(ctx EvalContext) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	actionVals, actionDiags := n.ActionConfig.EvalInstances(ctx, n.Addr.Action, nil)
	diags = diags.Append(actionDiags)
	if diags.HasErrors() {
		return diags
	}

	for key, actionVal := range actionVals.Iter() {
		diags = diags.Append(n.planAction(ctx, n.ActionConfig.Config, key.Absolute(ctx.Path()), actionVal))
	}

	return diags
}

func (n *nodeActionPlanInvoke) planAction(ctx EvalContext, config *configs.Action, addr addrs.AbsActionInstance, configVal cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	ai := plans.ActionInvocationInstance{
		Addr:          addr,
		ActionTrigger: new(plans.InvokeActionTrigger),
		ProviderAddr:  n.ProviderAddr,
		ConfigValue:   ephemeral.RemoveEphemeralValues(configVal),
	}

	provider, _, err := getProvider(ctx, n.ProviderAddr)
	if err != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to get provider",
			Detail:   fmt.Sprintf("Failed to get provider while triggering action %s: %s.", n.Addr, err),
			Subject:  config.DeclRange.Ptr(),
		})
	}

	unmarkedConfig, _ := configVal.UnmarkDeepWithPaths()

	if !unmarkedConfig.IsWhollyKnown() {
		// we're not actually planning or applying changes from the
		// configuration. if the configuration of the action has unknown values
		// it means one of the resources that are referenced hasn't actually
		// been created.
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Partially applied configuration",
			Detail:   fmt.Sprintf("The action %s contains unknown values while planning. This means it is referencing resources that have not yet been created, please run a complete plan/apply cycle to ensure the state matches the configuration before using the -invoke argument.", n.Addr),
			Subject:  config.DeclRange.Ptr(),
		})
	}

	resp := provider.PlanAction(providers.PlanActionRequest{
		ActionType:         addr.Action.Action.Type,
		ProposedActionData: unmarkedConfig,
		ClientCapabilities: ctx.ClientCapabilities(),
	})

	diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config, addr.ContainingAction().String()))
	if resp.Deferred != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider deferred an action",
			Detail:   fmt.Sprintf("The provider for %s ordered the action deferred. This likely means you are executing the action against a configuration that hasn't been completely applied.", n.Addr),
			Subject:  config.DeclRange.Ptr(),
		})
	}

	ctx.Changes().AppendActionInvocation(&ai)
	return diags
}

// nodeActionInvokeApplyInstance represents a single action instance to call,
// which was triggered via a manual invoke command.
type nodeActionInvokeApplyInstance struct {
	*actionTriggerApplyInstance
}

var (
	_ GraphNodeExecutable       = (*nodeActionInvokeApplyInstance)(nil)
	_ GraphNodeReferencer       = (*nodeActionInvokeApplyInstance)(nil)
	_ GraphNodeProviderConsumer = (*nodeActionInvokeApplyInstance)(nil)
	_ GraphNodeModulePath       = (*nodeActionInvokeApplyInstance)(nil)
)

func (n *nodeActionInvokeApplyInstance) Name() string {
	return n.ActionInvocation.Addr.String() + " (invoke)"
}

func (n *nodeActionInvokeApplyInstance) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	return n.invoke(ctx, op)
}

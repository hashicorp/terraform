// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

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

	// Callers is a list of resources which reference an action which uses the
	// caller symbol.
	Callers []addrs.ConfigResource
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
	// Callers are added to references so we can keep the caller nodes in the
	// graph. The instances must be evaluated from state, but evaluation
	// currently requires that resources at least be processed before
	// evaluation.
	var callers []*addrs.Reference
	for _, caller := range n.Callers {
		callers = append(callers, &addrs.Reference{
			Subject: caller.Resource,
		})
	}

	return append([]*addrs.Reference{
		{
			Subject: n.Addr.Action,
		},
		{
			Subject: n.Addr.Action.Action,
		},
	}, callers...)
}

func (n *nodeActionInvokeExpand) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var g Graph

	expander := ctx.InstanceExpander()

	// invoke only operates via the current state, so any callers must be looked
	// up via the state. The instances expander won't know about them, and there
	// will be no changes to find.
	syncState := ctx.State()
	state := syncState.Lock()
	defer syncState.Unlock()

	for _, mod := range expander.ExpandModule(n.Module, false) {
		if !mod.TargetContains(n.Target) {
			continue
		}

		if len(n.Callers) == 0 {
			g.Add(&nodeActionPlanInvoke{
				Module:       mod,
				Addr:         n.Addr,
				ActionConfig: n.ActionConfig,
				ProviderAddr: n.ActionConfig.ResolvedProvider,
			})
		} else {
			for _, caller := range n.Callers {
				for _, res := range state.Resources(caller) {
					if !mod.TargetContains(res.Addr) {
						// resource from the wrong module instance
						continue
					}

					for instKey, resInst := range res.Instances {
						if resInst.Current == nil {
							continue
						}

						log.Printf("[TRACE] expanding %s invoke node for caller %s", n.Addr, res.Addr.Resource)
						g.Add(&nodeActionPlanInvoke{
							Module:       mod,
							Addr:         n.Addr,
							ActionConfig: n.ActionConfig,
							ProviderAddr: n.ActionConfig.ResolvedProvider,
							Caller:       res.Addr.Resource.Instance(instKey),
						})
					}
				}
			}
		}
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
	Caller       addrs.Referenceable
}

func (n *nodeActionPlanInvoke) Name() string {
	return n.Addr.String()
}

func (n *nodeActionPlanInvoke) Path() addrs.ModuleInstance {
	return n.Module
}

func (n *nodeActionPlanInvoke) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
	// for now each action instance will be invoked serially
	return n.planActions(ctx)
}

func (n *nodeActionPlanInvoke) planActions(ctx EvalContext) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// We're relying on the given addr derived from the action target to
	// determine which action instance to evaluate. If the address has no key
	// and the action is expanded, we will plan all instances.
	actionVals, actionDiags := n.ActionConfig.EvalInvokedInstances(ctx, n.Addr.Action, n.Caller)
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

	var triggerAddr *addrs.AbsResourceInstance
	if n.Caller != nil {
		absCaller := n.Caller.(addrs.ResourceInstance).Absolute(ctx.Path())
		triggerAddr = &absCaller
	}

	ai := plans.ActionInvocationInstance{
		Addr: addr,
		ActionTrigger: &plans.InvokeActionTrigger{
			CallingResourceAddr: triggerAddr,
		},
		ProviderAddr: n.ProviderAddr,
		ConfigValue:  ephemeral.RemoveEphemeralValues(configVal),
		Caller:       n.Caller,
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
	var caller addrs.Referenceable

	switch trigger := n.ActionInvocation.ActionTrigger.(type) {
	case *plans.ResourceActionTrigger:
		caller = trigger.TriggeringResourceAddr.Resource
	case *plans.InvokeActionTrigger:
		if trigger.CallingResourceAddr != nil {
			caller = trigger.CallingResourceAddr.Resource
		}
	}

	return n.Invoke(ctx, caller, cty.NilVal)
}

func (n *nodeActionInvokeApplyInstance) References() []*addrs.Reference {
	refs := n.actionTriggerApplyInstance.References()

	// add any caller to ensure the resource expansion nodes remain in the graph
	switch trigger := n.ActionInvocation.ActionTrigger.(type) {
	case *plans.ResourceActionTrigger:
		refs = append(refs, &addrs.Reference{
			Subject: trigger.TriggeringResourceAddr.Resource,
		})
	case *plans.InvokeActionTrigger:
		if trigger.CallingResourceAddr != nil {
			refs = append(refs, &addrs.Reference{
				Subject: trigger.CallingResourceAddr.Resource,
			})
		}
	}

	return refs
}

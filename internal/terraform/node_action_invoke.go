// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/ephemeral"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	_ GraphNodeDynamicExpandable = (*nodeActionInvokeExpand)(nil)
	_ GraphNodeReferencer        = (*nodeActionInvokeExpand)(nil)
	_ GraphNodeProviderConsumer  = (*nodeActionInvokeExpand)(nil)
)

type nodeActionInvokeExpand struct {
	Target addrs.Targetable
	Config *configs.Action

	resolvedProvider addrs.AbsProviderConfig // set during the graph walk
}

func (n *nodeActionInvokeExpand) ProvidedBy() (addr addrs.ProviderConfig, exact bool) {
	// Once the provider is fully resolved, we can return the known value.
	if n.resolvedProvider.Provider.Type != "" {
		return n.resolvedProvider, true
	}

	// Since we always have a config, we can use it
	relAddr := n.Config.ProviderConfigAddr()
	return addrs.LocalProviderConfig{
		LocalName: relAddr.LocalName,
		Alias:     relAddr.Alias,
	}, false
}

func (n *nodeActionInvokeExpand) Provider() (provider addrs.Provider) {
	return n.Config.Provider
}

func (n *nodeActionInvokeExpand) SetProvider(p addrs.AbsProviderConfig) {
	n.resolvedProvider = p
}

func (n *nodeActionInvokeExpand) ModulePath() addrs.Module {
	switch target := n.Target.(type) {
	case addrs.AbsActionInstance:
		return target.Module.Module()
	case addrs.AbsAction:
		return target.Module.Module()
	default:
		panic("unrecognized action type")
	}
}

func (n *nodeActionInvokeExpand) References() []*addrs.Reference {
	switch target := n.Target.(type) {
	case addrs.AbsActionInstance:
		return []*addrs.Reference{
			{
				Subject: target.Action,
			},
			{
				Subject: target.Action.Action,
			},
		}
	case addrs.AbsAction:
		return []*addrs.Reference{
			{
				Subject: target.Action,
			},
		}
	default:
		panic("not an action target")
	}
}

func (n *nodeActionInvokeExpand) DynamicExpand(context EvalContext) (*Graph, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if n.Config == nil {
		// This means the user specified an action target that does not exist.
		return nil, diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid action target",
			fmt.Sprintf("Action %s does not exist within the configuration.", n.Target.String())))
	}

	var g Graph
	switch addr := n.Target.(type) {
	case addrs.AbsActionInstance:
		if _, ok := context.Actions().GetActionInstance(addr); !ok {
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid action",
				Detail:   fmt.Sprintf("Targeted action does not exist after expansion: %s.", addr),
				Subject:  n.Config.DeclRange.Ptr(),
			})
		} else {
			g.Add(&nodeActionInvokeInstance{
				Target: addr,
				Config: n.Config,
			})
		}

	case addrs.AbsAction:
		for _, target := range context.Actions().GetActionInstanceKeys(addr) {
			g.Add(&nodeActionInvokeInstance{
				Target: target,
				Config: n.Config,
			})
		}
	}
	addRootNodeToGraph(&g)
	return &g, diags
}

var (
	_ GraphNodeExecutable     = (*nodeActionInvokeInstance)(nil)
	_ GraphNodeModuleInstance = (*nodeActionInvokeInstance)(nil)
)

type nodeActionInvokeInstance struct {
	Target addrs.AbsActionInstance
	Config *configs.Action
}

func (n *nodeActionInvokeInstance) Path() addrs.ModuleInstance {
	return n.Target.Module
}

func (n *nodeActionInvokeInstance) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	actionInstance, ok := ctx.Actions().GetActionInstance(n.Target)
	if !ok {
		// shouldn't happen, we checked these things exist in the expand node
		panic("tried to trigger non-existent action")
	}

	ai := plans.ActionInvocationInstance{
		Addr:          n.Target,
		ActionTrigger: new(plans.InvokeActionTrigger),
		ProviderAddr:  actionInstance.ProviderAddr,
		ConfigValue:   ephemeral.RemoveEphemeralValues(actionInstance.ConfigValue),
	}

	provider, _, err := getProvider(ctx, actionInstance.ProviderAddr)
	if err != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to get provider",
			Detail:   fmt.Sprintf("Failed to get provider while triggering action %s: %s.", n.Target, err),
			Subject:  n.Config.DeclRange.Ptr(),
		})
	}

	unmarkedConfig, _ := actionInstance.ConfigValue.UnmarkDeepWithPaths()

	if !unmarkedConfig.IsWhollyKnown() {
		// we're not actually planning or applying changes from the
		// configuration. if the configuration of the action has unknown values
		// it means one of the resources that are referenced hasn't actually
		// been created.
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Partially applied configuration",
			Detail:   fmt.Sprintf("The action %s contains unknown values while planning. This means it is referencing resources that have not yet been created, please run a complete plan/apply cycle to ensure the state matches the configuration before using the -invoke argument.", n.Target.String()),
			Subject:  n.Config.DeclRange.Ptr(),
		})
	}

	resp := provider.PlanAction(providers.PlanActionRequest{
		ActionType:         n.Target.Action.Action.Type,
		ProposedActionData: unmarkedConfig,
		ClientCapabilities: ctx.ClientCapabilities(),
	})

	diags = diags.Append(resp.Diagnostics.InConfigBody(n.Config.Config, n.Target.ContainingAction().String()))
	if resp.Deferred != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider deferred an action",
			Detail:   fmt.Sprintf("The provider for %s ordered the action deferred. This likely means you are executing the action against a configuration that hasn't been completely applied.", n.Target),
			Subject:  n.Config.DeclRange.Ptr(),
		})
	}

	ctx.Changes().AppendActionInvocation(&ai)
	return diags
}

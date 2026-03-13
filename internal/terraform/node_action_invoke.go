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
	_ GraphNodeDynamicExpandable = (*nodeActionInvokeExpand)(nil)
	_ GraphNodeReferencer        = (*nodeActionInvokeExpand)(nil)
	_ GraphNodeProviderConsumer  = (*nodeActionInvokeExpand)(nil)
)

type nodeActionInvokeExpand struct {
	TargetAction addrs.Targetable
	Config       *configs.Action

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
	switch target := n.TargetAction.(type) {
	case addrs.AbsActionInstance:
		return target.Module.Module()
	case addrs.AbsAction:
		return target.Module.Module()
	default:
		panic("unrecognized action type")
	}
}

func (n *nodeActionInvokeExpand) References() []*addrs.Reference {
	switch target := n.TargetAction.(type) {
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

func (n *nodeActionInvokeExpand) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if n.Config == nil {
		// This means the user specified an action target that does not exist.
		return nil, diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid action target",
			fmt.Sprintf("Action %s does not exist within the configuration.", n.TargetAction.String())))
	}

	var g Graph
	switch addr := n.TargetAction.(type) {
	case addrs.AbsActionInstance:
		g.Add(&nodeActionInvokeInstance{
			TargetAction:     addr,
			Config:           n.Config,
			resolvedProvider: n.resolvedProvider,
		})
	case addrs.AbsAction:
		instances := ctx.InstanceExpander().ExpandAction(addr)
		for _, target := range instances {
			g.Add(&nodeActionInvokeInstance{
				TargetAction:     target,
				Config:           n.Config,
				resolvedProvider: n.resolvedProvider,
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
	TargetAction addrs.AbsActionInstance
	Config       *configs.Action

	resolvedProvider addrs.AbsProviderConfig
}

func (n *nodeActionInvokeInstance) Path() addrs.ModuleInstance {
	return n.TargetAction.Module
}

func (n *nodeActionInvokeInstance) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	provider, schema, err := getProvider(ctx, n.resolvedProvider)
	if err != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to get provider",
			Detail:   fmt.Sprintf("Failed to get provider while triggering action %s: %s.", n.TargetAction, err),
			Subject:  n.Config.DeclRange.Ptr(),
		})
	}
	actionSchema := schema.Actions[n.Config.Type]

	// get the action expansion and config for evaluation
	allInsts := ctx.InstanceExpander()
	keyData := allInsts.GetActionInstanceRepetitionData(n.TargetAction)

	configVal := cty.NullVal(actionSchema.ConfigSchema.ImpliedType())
	if n.Config.Config != nil {
		var configDiags tfdiags.Diagnostics
		configVal, _, configDiags = ctx.EvaluateBlock(n.Config.Config, actionSchema.ConfigSchema, nil, keyData)

		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			return diags
		}

		valDiags := validateResourceForbiddenEphemeralValues(ctx, configVal, actionSchema.ConfigSchema)
		diags = diags.Append(valDiags.InConfigBody(n.Config.Config, n.TargetAction.String()))
		if valDiags.HasErrors() {
			return diags
		}
		_, deprecationDiags := ctx.Deprecations().ValidateAndUnmarkConfig(configVal, actionSchema.ConfigSchema, n.TargetAction.ConfigAction().Module)
		diags = diags.Append(deprecationDiags.InConfigBody(n.Config.Config, n.TargetAction.String()))
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
			Detail:   fmt.Sprintf("The action %s contains unknown values while planning. This means it is referencing resources that have not yet been created, please run a complete plan/apply cycle to ensure the state matches the configuration before using the -invoke argument.", n.TargetAction.String()),
			Subject:  n.Config.DeclRange.Ptr(),
		})
	}

	resp := provider.PlanAction(providers.PlanActionRequest{
		ActionType:         n.TargetAction.Action.Action.Type,
		ProposedActionData: unmarkedConfig,
		ClientCapabilities: ctx.ClientCapabilities(),
	})

	diags = diags.Append(resp.Diagnostics.InConfigBody(n.Config.Config, n.TargetAction.ContainingAction().String()))
	if resp.Deferred != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider deferred an action",
			Detail:   fmt.Sprintf("The provider for %s ordered the action deferred. This likely means you are executing the action against a configuration that hasn't been completely applied.", n.TargetAction),
			Subject:  n.Config.DeclRange.Ptr(),
		})
	}

	ai := plans.ActionInvocationInstance{
		Addr:          n.TargetAction,
		ActionTrigger: new(plans.InvokeActionTrigger),
		ProviderAddr:  n.resolvedProvider,
		ConfigValue:   ephemeral.RemoveEphemeralValues(configVal),
	}

	ctx.Changes().AppendActionInvocation(&ai)
	return diags
}

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

// nodeActionInvokeAbstract represents an action invocation that has no associated
// operations.
type nodeActionInvokeAbstract struct {
	Target           addrs.Targetable
	Config           *configs.Action
	resolvedProvider addrs.AbsProviderConfig // set during the graph walk
	Schema           *providers.ActionSchema
}

var (
	_ GraphNodeReferencer         = (*nodeActionInvokeAbstract)(nil)
	_ GraphNodeProviderConsumer   = (*nodeActionInvokeAbstract)(nil)
	_ GraphNodeAttachActionSchema = (*nodeActionInvokeAbstract)(nil)
	_ GraphNodeProviderConsumer   = (*nodeActionInvokeAbstract)(nil)
	_ GraphNodeReferencer         = (*nodeActionInvokeAbstract)(nil)
)

func (n *nodeActionInvokeAbstract) Name() string {
	invoke := " (invoke)"
	switch target := n.Target.(type) {
	case addrs.AbsActionInstance:
		return target.ConfigAction().String() + invoke
	case addrs.AbsAction:
		return target.Action.InModule(target.Module.Module()).String() + invoke
	default:
		panic("unrecognized action type")
	}
}

func (n *nodeActionInvokeAbstract) ActionAddr() addrs.ConfigAction {
	switch target := n.Target.(type) {
	case addrs.AbsActionInstance:
		return target.ConfigAction()
	case addrs.AbsAction:
		return target.Action.InModule(target.Module.Module())
	default:
		panic(fmt.Sprintf("unrecognized action type %s", target))
	}
}

func (n *nodeActionInvokeAbstract) AttachActionSchema(schema *providers.ActionSchema) {
	n.Schema = schema
}

func (n *nodeActionInvokeAbstract) ProvidedBy() (addr addrs.ProviderConfig, exact bool) {
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

func (n *nodeActionInvokeAbstract) Provider() (provider addrs.Provider) {
	return n.Config.Provider
}

func (n *nodeActionInvokeAbstract) SetProvider(p addrs.AbsProviderConfig) {
	n.resolvedProvider = p
}

func (n *nodeActionInvokeAbstract) ModulePath() addrs.Module {
	switch target := n.Target.(type) {
	case addrs.AbsActionInstance:
		return target.Module.Module()
	case addrs.AbsAction:
		return target.Module.Module()
	default:
		panic("unrecognized action type")
	}
}

func (n *nodeActionInvokeAbstract) References() []*addrs.Reference {
	var refs []*addrs.Reference

	switch target := n.Target.(type) {
	case addrs.AbsActionInstance:
		refs = append(refs, []*addrs.Reference{
			{Subject: target.Action},
			{Subject: target.Action.Action},
		}...)
	case addrs.AbsAction:
		refs = append(refs, &addrs.Reference{Subject: target.Action})
	default:
		panic("not an action target")
	}

	return refs
}

var (
	_ GraphNodeDynamicExpandable = (*nodeActionInvokeExpand)(nil)
	_ GraphNodeReferencer        = (*nodeActionInvokeExpand)(nil)
)

type nodeActionInvokeExpand struct {
	nodeActionInvokeAbstract
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

	allInsts := context.InstanceExpander().AllInstances()
	var g Graph
	switch addr := n.Target.(type) {
	case addrs.AbsActionInstance:
		if !allInsts.HasActionInstance(addr) {
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid action",
				Detail:   fmt.Sprintf("Targeted action does not exist after expansion: %s.", addr),
				Subject:  n.Config.DeclRange.Ptr(),
			})
		} else {
			g.Add(&nodeActionInvokePlanInstance{
				nodeActionInvokeAbstract: n.nodeActionInvokeAbstract,
				ActionTarget:             addr,
			})
		}

	case addrs.AbsAction:
		instances := context.InstanceExpander().ExpandAction(addr)
		for _, target := range instances {
			g.Add(&nodeActionInvokePlanInstance{
				nodeActionInvokeAbstract: n.nodeActionInvokeAbstract,
				ActionTarget:             target,
			})
		}
	}
	addRootNodeToGraph(&g)
	return &g, diags
}

var (
	_ GraphNodeExecutable     = (*nodeActionInvokePlanInstance)(nil)
	_ GraphNodeModuleInstance = (*nodeActionInvokePlanInstance)(nil)
	_ GraphNodeReferencer     = (*nodeActionInvokePlanInstance)(nil)
)

type nodeActionInvokePlanInstance struct {
	nodeActionInvokeAbstract
	ActionTarget addrs.AbsActionInstance
}

func (n *nodeActionInvokePlanInstance) Path() addrs.ModuleInstance {
	return n.ActionTarget.Module
}

func (n *nodeActionInvokePlanInstance) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	allInsts := ctx.InstanceExpander()
	keyData := allInsts.GetActionInstanceRepetitionData(n.ActionTarget)

	configVal := cty.NullVal(n.Schema.ConfigSchema.ImpliedType())
	if n.Config.Config != nil {
		var configDiags, deprecationDiags tfdiags.Diagnostics
		configVal, _, configDiags = ctx.EvaluateBlock(n.Config.Config, n.Schema.ConfigSchema, nil, keyData)
		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			return diags
		}

		valDiags := validateResourceForbiddenEphemeralValues(ctx, configVal, n.Schema.ConfigSchema)
		diags = diags.Append(valDiags.InConfigBody(n.Config.Config, n.ActionTarget.String()))

		configVal, deprecationDiags = ctx.Deprecations().ValidateConfig(configVal, n.Schema.ConfigSchema, n.ModulePath())
		diags = diags.Append(deprecationDiags.InConfigBody(n.Config.Config, n.ActionTarget.String()))

		if diags.HasErrors() {
			return diags
		}
	}

	ai := plans.ActionInvocationInstance{
		Addr:          n.ActionTarget,
		ActionTrigger: new(plans.InvokeActionTrigger),
		ProviderAddr:  n.resolvedProvider,
		ConfigValue:   ephemeral.RemoveEphemeralValues(configVal),
	}

	provider, _, err := getProvider(ctx, n.resolvedProvider)
	if err != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to get provider",
			Detail:   fmt.Sprintf("Failed to get provider while triggering action %s: %s.", n.Target, err),
			Subject:  n.Config.DeclRange.Ptr(),
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
			Detail:   fmt.Sprintf("The action %s contains unknown values while planning. This means it is referencing resources that have not yet been created, please run a complete plan/apply cycle to ensure the state matches the configuration before using the -invoke argument.", n.Target.String()),
			Subject:  n.Config.DeclRange.Ptr(),
		})
	}

	resp := provider.PlanAction(providers.PlanActionRequest{
		ActionType:         n.ActionTarget.Action.Action.Type,
		ProposedActionData: unmarkedConfig,
		ClientCapabilities: ctx.ClientCapabilities(),
	})

	diags = diags.Append(resp.Diagnostics.InConfigBody(n.Config.Config, n.ActionTarget.ContainingAction().String()))
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

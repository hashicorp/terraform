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

// no graph functions, just interface implementations
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
)

func (n *nodeActionInvokeAbstract) Name() string {
	switch target := n.Target.(type) {
	case addrs.AbsActionInstance:
		return target.ConfigAction().String()
	case addrs.AbsAction:
		return target.Action.InModule(target.Module.Module()).String()
	default:
		panic("unrecognized action type")
	}
}

func (n *nodeActionInvokeAbstract) ActionAddr() addrs.ConfigAction {
	if n.Target == nil {
		panic("nil target??")
	}
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
			{
				Subject: target.Action,
			},
			{
				Subject: target.Action.Action,
			},
		}...)
	case addrs.AbsAction:
		refs = append(refs, &addrs.Reference{Subject: target.Action})
	default:
		panic("not an action target")
	}

	c := n.Config
	countRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, c.Count)
	refs = append(refs, countRefs...)
	forEachRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, c.ForEach)
	refs = append(refs, forEachRefs...)

	if n.Schema != nil {
		configRefs, _ := langrefs.ReferencesInBlock(addrs.ParseRef, c.Config, n.Schema.ConfigSchema)
		refs = append(refs, configRefs...)
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

func (n *nodeActionInvokeExpand) Name() string {
	return n.nodeActionInvokeAbstract.Name()
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
			g.Add(&nodeActionInvokePlanInstance{
				nodeActionInvokeAbstract: n.nodeActionInvokeAbstract,
				ActionTarget:             addr,
			})
		}

	case addrs.AbsAction:
		for _, target := range context.Actions().GetActionInstanceKeys(addr) {
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
)

type nodeActionInvokePlanInstance struct {
	nodeActionInvokeAbstract
	ActionTarget addrs.AbsActionInstance
}

func (n *nodeActionInvokePlanInstance) Name() string {
	return n.ActionTarget.String() + " (instance)"
}

func (n *nodeActionInvokePlanInstance) ActionAddr() addrs.ConfigAction {
	return n.ActionTarget.ConfigAction()
}

func (n *nodeActionInvokePlanInstance) Path() addrs.ModuleInstance {
	return n.ActionTarget.Module
}

func (n *nodeActionInvokePlanInstance) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	actionInstance, ok := ctx.Actions().GetActionInstance(n.ActionTarget)
	if !ok {
		// shouldn't happen, we checked these things exist in the expand node
		panic("tried to trigger non-existent action")
	}

	ai := plans.ActionInvocationInstance{
		Addr:          n.ActionTarget,
		ActionTrigger: new(plans.InvokeActionTrigger),
		ProviderAddr:  actionInstance.ProviderAddr,
		ConfigValue:   ephemeral.RemoveEphemeralValues(actionInstance.ConfigValue),
	}

	provider, _, err := getProvider(ctx, actionInstance.ProviderAddr)
	if err != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to get provider",
			Detail:   fmt.Sprintf("Failed to get provider while triggering action %s: %s.", n.ActionTarget, err),
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
			Detail:   fmt.Sprintf("The action %s contains unknown values while planning. This means it is referencing resources that have not yet been created, please run a complete plan/apply cycle to ensure the state matches the configuration before using the -invoke argument.", n.ActionTarget.String()),
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
			Detail:   fmt.Sprintf("The provider for %s ordered the action deferred. This likely means you are executing the action against a configuration that hasn't been completely applied.", n.ActionTarget),
			Subject:  n.Config.DeclRange.Ptr(),
		})
	}

	ctx.Changes().AppendActionInvocation(&ai)
	return diags
}

type nodeActionInvokeApplyInstance struct {
	nodeActionInvokeAbstract
	ActionInvocation *plans.ActionInvocationInstanceSrc
	resolvedProvider addrs.AbsProviderConfig
}

var (
	_ GraphNodeExecutable       = (*nodeActionInvokeApplyInstance)(nil)
	_ GraphNodeReferencer       = (*nodeActionInvokeApplyInstance)(nil)
	_ GraphNodeProviderConsumer = (*nodeActionInvokeApplyInstance)(nil)
	_ GraphNodeModulePath       = (*nodeActionInvokeApplyInstance)(nil)
)

func (n *nodeActionInvokeApplyInstance) Name() string {
	return n.ActionInvocation.Addr.String() + " (instance)"
}

func (n *nodeActionInvokeApplyInstance) Execute(ctx EvalContext, wo walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	actionInvocation := n.ActionInvocation

	ai := ctx.Changes().GetActionInvocation(actionInvocation.Addr, actionInvocation.ActionTrigger)
	if ai == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Action invocation not found in plan",
			Detail:   "Could not find action invocation for address " + actionInvocation.Addr.String(),
		})
		return diags
	}
	actionData, ok := ctx.Actions().GetActionInstance(ai.Addr)
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Action instance not found",
			Detail:   "Could not find action instance for address " + ai.Addr.String(),
		})
		return diags
	}
	provider, schema, err := getProvider(ctx, actionData.ProviderAddr)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Failed to get provider for %s", ai.Addr),
			Detail:   fmt.Sprintf("Failed to get provider: %s", err),
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
		})
	}

	if !configValue.IsWhollyKnown() {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Action configuration unknown during apply",
			Detail:   fmt.Sprintf("The action %s was not fully known during apply.\n\nThis is a bug in Terraform, please report it.", ai.Addr.Action.String()),
		})
	}

	hookIdentity := HookActionIdentity{
		Addr:          ai.Addr,
		ActionTrigger: ai.ActionTrigger,
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
		ActionType:         ai.Addr.Action.Action.Type,
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
		})
	}

	return diags
}

// mildwonkey: how did this work for invoke/target?? just leave the range blank?
// not sure if I should drop this or not.
func (n *nodeActionInvokeApplyInstance) AddSubjectToDiagnostics(input tfdiags.Diagnostics) tfdiags.Diagnostics {
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
		})
	}
	return diags
}

func (n *nodeActionInvokeApplyInstance) ProvidedBy() (addr addrs.ProviderConfig, exact bool) {
	return n.ActionInvocation.ProviderAddr, true
}

func (n *nodeActionInvokeApplyInstance) Provider() (provider addrs.Provider) {
	return n.ActionInvocation.ProviderAddr.Provider
}

func (n *nodeActionInvokeApplyInstance) SetProvider(config addrs.AbsProviderConfig) {
	n.resolvedProvider = config
}

func (n *nodeActionInvokeApplyInstance) References() []*addrs.Reference {
	refs := []*addrs.Reference{{Subject: n.ActionInvocation.Addr.Action}}

	c := n.Config
	countRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, c.Count)
	refs = append(refs, countRefs...)
	forEachRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, c.ForEach)
	refs = append(refs, forEachRefs...)

	if n.Schema != nil {
		configRefs, _ := langrefs.ReferencesInBlock(addrs.ParseRef, c.Config, n.Schema.ConfigSchema)
		refs = append(refs, configRefs...)
	}

	return refs
}

func (n *nodeActionInvokeApplyInstance) ModulePath() addrs.Module {
	return n.ActionInvocation.Addr.Module.Module()
}

func (n *nodeActionInvokeApplyInstance) Path() addrs.ModuleInstance {
	return n.ActionInvocation.Addr.Module
}

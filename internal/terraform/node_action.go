// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// GraphNodeConfigAction is implemented by any nodes that represent an action.
// The type of operation cannot be assumed, only that this node represents
// the given resource.
type GraphNodeConfigAction interface {
	ActionAddr() addrs.ConfigAction
}

// NodeActionConfig represents an action in the configuration. This node is
// primarily concerned with resolving provider references and receiving the
// correct schema. All expansion and execution is done from an action trigger.
type NodeActionConfig struct {
	Addr addrs.ConfigAction

	Config *configs.Action

	// The fields below will be automatically set using the Attach interfaces if
	// you're running those transforms, but also can be explicitly set if you
	// already have that information.

	// The address of the provider this action will use
	ResolvedProvider addrs.AbsProviderConfig
	Schema           *providers.ActionSchema
	Dependencies     []addrs.ConfigResource
}

var (
	_ GraphNodeReferenceable      = (*NodeActionConfig)(nil)
	_ GraphNodeReferencer         = (*NodeActionConfig)(nil)
	_ GraphNodeConfigAction       = (*NodeActionConfig)(nil)
	_ GraphNodeAttachActionSchema = (*NodeActionConfig)(nil)
	_ GraphNodeProviderConsumer   = (*NodeActionConfig)(nil)
	_ GraphNodeAttachDependencies = (*NodeActionConfig)(nil)
)

func (n NodeActionConfig) Name() string {
	return n.Addr.String()
}

func (n *NodeActionConfig) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	// Validation happens without expansion, and Validate will be called from
	// either a validateable node or a triggering node.
	if op == walkValidate {
		return nil
	}

	// Action configuration is always evaluated from the context of the
	// triggering node, so all this node needs to do for Execute is record the
	// instance expansion. This also makes sure we determine whether we need
	// to be deferred due to unknown expansion before we get to the resources
	// triggering the action.
	return n.recordActionExpansion(ctx)
}

func (n *NodeActionConfig) recordActionExpansion(ctx EvalContext) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// FIXME: this is hard-coded to false for now to match existing behavior,
	// but actions will need to conform to the same deferral system as all other
	// objects.
	// deferralAllowed := ctx.Deferrals().DeferralAllowed()
	deferralAllowed := false

	expander := ctx.InstanceExpander()
	for _, module := range expander.ExpandModule(n.Addr.Module, false) {
		moduleCtx := evalContextForModuleInstance(ctx, module)

		switch {
		case n.Config.Count != nil:
			count, countDiags := evaluateCountExpression(n.Config.Count, moduleCtx, deferralAllowed)
			diags = diags.Append(countDiags)
			if diags.HasErrors() {
				return diags
			}
			if count >= 0 {
				expander.SetActionCount(module, n.Addr.Action, count)

			} else {
				expander.SetActionCountUnknown(module, n.Addr.Action)
			}

		case n.Config.ForEach != nil:
			forEach, known, forEachDiags := evaluateForEachExpression(n.Config.ForEach, moduleCtx, deferralAllowed)
			diags = diags.Append(forEachDiags)
			if forEachDiags.HasErrors() {
				return diags
			}
			if known {
				expander.SetActionForEach(module, n.Addr.Action, forEach)
			} else {
				expander.SetActionForEachUnknown(module, n.Addr.Action)
			}

		default:
			expander.SetActionSingle(module, n.Addr.Action)
		}
	}

	return diags
}

// Validate validates the action config, with an optional caller address if the
// action is invoked from a resource action trigger.
func (n *NodeActionConfig) Validate(ctx EvalContext, caller addrs.Referenceable) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	keyData := EvalDataForNoInstanceKey

	switch {
	case n.Config.Count != nil:
		// If the config block has count, we'll evaluate with an unknown
		// number as count.index so we can still type check even though
		// we won't expand count until the plan phase.
		keyData = InstanceKeyEvalData{
			CountIndex: cty.UnknownVal(cty.Number),
		}

		// Basic type-checking of the count argument. More complete validation
		// of this will happen when we DynamicExpand during the plan walk.
		_, countDiags := evaluateCountExpressionValue(n.Config.Count, ctx)
		diags = diags.Append(countDiags)

	case n.Config.ForEach != nil:
		keyData = InstanceKeyEvalData{
			EachKey:   cty.UnknownVal(cty.String),
			EachValue: cty.UnknownVal(cty.DynamicPseudoType),
		}

		// Evaluate the for_each expression here so we can expose the diagnostics
		forEachDiags := newForEachEvaluator(n.Config.ForEach, ctx, false).ValidateActionValue()
		diags = diags.Append(forEachDiags)
	}

	schema := providerSchema.SchemaForActionType(n.Config.Type)
	if schema.ConfigSchema == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid action type",
			Detail:   fmt.Sprintf("The provider %s does not support action type %q.", n.Provider().ForDisplay(), n.Config.Type),
			Subject:  &n.Config.TypeRange,
		})
		return diags
	}

	config := n.Config.Config
	if n.Config.Config == nil {
		config = hcl.EmptyBody()
	}

	configVal, _, valDiags := ctx.EvaluateBlock(config, schema.ConfigSchema, caller, keyData)
	if valDiags.HasErrors() {
		// If there was no config block at all, we'll add a Context range to the returned diagnostic
		if n.Config.Config == nil {
			for _, diag := range valDiags.ToHCL() {
				diag.Context = &n.Config.DeclRange
				diags = diags.Append(diag)
			}
			return diags
		} else {
			diags = diags.Append(valDiags)
			return diags
		}
	}
	var deprecationDiags tfdiags.Diagnostics
	configVal, deprecationDiags = ctx.Deprecations().ValidateAndUnmarkConfig(configVal, schema.ConfigSchema, n.ModulePath())
	diags = diags.Append(deprecationDiags.InConfigBody(n.Config.Config, n.Addr.String()))

	valDiags = validateResourceForbiddenEphemeralValues(ctx, configVal, schema.ConfigSchema)
	diags = diags.Append(valDiags.InConfigBody(config, n.Addr.String()))

	if diags.HasErrors() {
		return diags
	}

	// Use unmarked value for validate request
	unmarkedConfigVal, _ := configVal.UnmarkDeep()
	log.Printf("[TRACE] Validating config for %q", n.Addr)
	req := providers.ValidateActionConfigRequest{
		TypeName: n.Config.Type,
		Config:   unmarkedConfigVal,
	}

	resp := provider.ValidateActionConfig(req)
	diags = diags.Append(resp.Diagnostics.InConfigBody(n.Config.Config, n.Addr.String()))

	return diags
}

// ConcreteActionNodeFunc is a callback type used to convert an
// abstract action to a concrete one of some type.
type ConcreteActionNodeFunc func(*NodeActionConfig) dag.Vertex

// GraphNodeConfigAction
func (n NodeActionConfig) ActionAddr() addrs.ConfigAction {
	return n.Addr
}

func (n NodeActionConfig) ModulePath() addrs.Module {
	return n.Addr.Module
}

func (n *NodeActionConfig) Path() addrs.ModuleInstance {
	// this node is only directly evaluated during validation, so there is never
	// module expansion.
	return n.Addr.Module.UnkeyedInstanceShim()
}

func (n *NodeActionConfig) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr.Action}
}

func (n *NodeActionConfig) References() []*addrs.Reference {
	var result []*addrs.Reference
	c := n.Config

	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, c.Count)
	result = append(result, refs...)
	refs, _ = langrefs.ReferencesInExpr(addrs.ParseRef, c.ForEach)
	result = append(result, refs...)

	if n.Schema != nil {
		refs, _ = langrefs.ReferencesInBlock(addrs.ParseRef, c.Config, n.Schema.ConfigSchema)
		result = append(result, refs...)
	}

	return result
}

func (n *NodeActionConfig) AttachActionSchema(schema *providers.ActionSchema) {
	n.Schema = schema
}

func (n *NodeActionConfig) Provider() ProviderRef {
	// If the resolvedProvider is set, use that
	if n.ResolvedProvider.Provider.Type != "" {
		ref := ProviderRef{
			Addr:     n.ResolvedProvider,
			Resolved: true,
		}
		return ref
	}

	var addr addrs.AbsProviderConfig
	if n.Config.Provider.Type != "" {
		addr.Provider = n.Config.Provider
	} else {
		addr.Provider = addrs.ImpliedProviderForUnqualifiedType(n.Addr.Action.ImpliedProvider())
	}

	addr.Alias = n.Config.ProviderConfigAddr().Alias
	addr.Module = n.ModulePath()
	return ProviderRef{
		Addr: addr,
	}
}

func (n *NodeActionConfig) SetProvider(p addrs.AbsProviderConfig) {
	n.ResolvedProvider = p
}

func (n *NodeActionConfig) AttachDependencies(deps []addrs.ConfigResource) {
	n.Dependencies = deps
}

// The invoke command can reference an action block to invoke all instances, so
// here we return a value representing the entire block if we have an
// addrs.NoKey This function uses addrs.ActionInstance even though it only needs
// the key because we need to use use a full instance addr for the resulting map
// keys anyway.
func (n *NodeActionConfig) EvalInstances(ctx EvalContext, addr addrs.ActionInstance, callRange *hcl.Range, caller addrs.Referenceable) (addrs.Map[addrs.ActionInstance, cty.Value], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	all := addrs.MakeMap[addrs.ActionInstance, cty.Value]()

	var instances []addrs.AbsActionInstance

	expander := ctx.InstanceExpander()
	if addr.Key == addrs.NoKey {
		// this might be a single instance with no key, or all instances, so we
		// must expand for both cases
		instances = expander.ExpandAction(n.Addr.Absolute(ctx.Path()))
	} else {
		// definitely looking for a single instance because we have an index of
		// some sort
		instances = []addrs.AbsActionInstance{addr.Absolute(ctx.Path())}
	}

	for _, instAddr := range instances {
		repData := expander.GetActionInstanceRepetitionData(instAddr)
		val, evalDiags := n.evalInstance(ctx, repData, callRange, caller)
		diags = diags.Append(evalDiags)
		if diags.HasErrors() {
			return all, diags
		}
		all.Put(instAddr.Action, val)
	}

	return all, diags
}

// EvalInstance returns the value from the expanded action block
func (n *NodeActionConfig) EvalInstance(ctx EvalContext, inst addrs.AbsActionInstance, callRange *hcl.Range, caller addrs.Referenceable) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	key := inst.Action.Key

	// we first validate the correct type of key is being used for the action
	switch {
	case n.Config.Count != nil:
		switch key := key.(type) {
		case addrs.IntKey:
			// OK

		case addrs.StringKey:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid index",
				Detail:   fmt.Sprintf("Invalid string key %s for action with count", key),
				Subject:  callRange,
			})
			return cty.DynamicVal, diags
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing index",
				Detail:   "An action with count must be referenced via an integer key",
				Subject:  callRange,
			})
			return cty.DynamicVal, diags
		}

	case n.Config.ForEach != nil:
		switch key := key.(type) {
		case addrs.IntKey:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid index",
				Detail:   fmt.Sprintf("Invalid key %d for action with for_each", key),
				Subject:  callRange,
			})
			return cty.DynamicVal, diags

		case addrs.StringKey:
			// OK

		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing index",
				Detail:   "An action with for_each must be referenced via a string key",
				Subject:  callRange,
			})
			return cty.DynamicVal, diags
		}

	default:
		switch key := key.(type) {
		case addrs.IntKey:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid index",
				Detail:   fmt.Sprintf("Unexpanded action referenced with instance key %d", key),
				Subject:  callRange,
			})

			return cty.DynamicVal, diags

		case addrs.StringKey:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid index",
				Detail:   fmt.Sprintf("Unexpanded action referenced with instance key %s", key),
				Subject:  callRange,
			})
			return cty.DynamicVal, diags
		}
	}

	instAddr := n.Addr.Absolute(ctx.Path()).Instance(key)

	expander := ctx.InstanceExpander()
	// first we have to make sure the instance is valid because the expander only panics
	instances := expander.ExpandAction(inst.ContainingAction())
	found := false
	for _, instAddr := range instances {
		if instAddr.Equal(inst) {
			found = true
		}
	}
	if !found {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to non-existent action instance",
			Detail:   fmt.Sprintf("The given key %s does not identify an instance of action.test_action.hello", key),
			Subject:  callRange,
		})
		return cty.DynamicVal, diags
	}

	repData := expander.GetActionInstanceRepetitionData(instAddr)

	return n.evalInstance(ctx, repData, callRange, caller)
}

// Eval one or more instances of the action. This function expects that the key
// is already validated for the the calling context, and will not produce
// diagnostics for incorrect key types.
func (n *NodeActionConfig) evalInstance(ctx EvalContext, repData instances.RepetitionData, callRange *hcl.Range, caller addrs.Referenceable) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// This should have been caught already
	if n.Schema == nil {
		panic("action eval called without a schema")
	}

	configVal := cty.NullVal(n.Schema.ConfigSchema.ImpliedType())
	if n.Config.Config != nil {
		var configDiags tfdiags.Diagnostics
		configVal, _, configDiags = ctx.EvaluateBlock(n.Config.Config, n.Schema.ConfigSchema.DeepCopy(), caller, repData)
		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			return configVal, diags
		}

		valDiags := validateResourceForbiddenEphemeralValues(ctx, configVal, n.Schema.ConfigSchema)
		diags = diags.Append(valDiags.InConfigBody(n.Config.Config, n.Addr.String()))

		var deprecationDiags tfdiags.Diagnostics
		configVal, deprecationDiags = ctx.Deprecations().ValidateAndUnmarkConfig(configVal, n.Schema.ConfigSchema, n.ModulePath())
		diags = diags.Append(deprecationDiags.InConfigBody(n.Config.Config, n.Addr.String()))
	}
	return configVal, diags
}

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

// The action config does not expand or execute itself during plan or apply, but
// for Validate it does verify valid configuration.
func (n *NodeActionConfig) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	if op != walkValidate {
		return nil
	}
	return n.validate(ctx)
}

func (n *NodeActionConfig) validate(ctx EvalContext) tfdiags.Diagnostics {
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

	configVal, _, valDiags := ctx.EvaluateBlock(config, schema.ConfigSchema, nil, keyData)
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

func (n *NodeActionConfig) repetitionData(ctx EvalContext) ([]instances.RepetitionData, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var reps []instances.RepetitionData

	switch {
	case n.Config.Count != nil:
		count, countDiags := evaluateCountExpression(n.Config.Count, ctx, false)
		diags = diags.Append(countDiags)
		if diags.HasErrors() {
			return nil, diags
		}

		for i := 0; i < count; i++ {
			reps = append(reps, instances.RepetitionData{
				CountIndex: cty.NumberIntVal(int64(i)),
			})
		}
		return reps, diags

	case n.Config.ForEach != nil:
		forEach, _, forEachDiags := evaluateForEachExpression(n.Config.ForEach, ctx, false)
		diags = diags.Append(forEachDiags)
		if forEachDiags.HasErrors() {
			return reps, diags
		}

		for key, value := range forEach {
			reps = append(reps, instances.RepetitionData{
				EachKey:   cty.StringVal(key),
				EachValue: value,
			})
		}
		return reps, diags

	default:
		return []instances.RepetitionData{EvalDataForNoInstanceKey}, diags
	}
}

// The invoke command can reference an action block to invoke all instances, so
// here we return a value representing the entire block if we have an
// addrs.NoKey This function uses addrs.ActionInstance even though it only needs
// the key because we need to use use a full instance addr for the resulting map
// keys anyway.
func (n *NodeActionConfig) EvalInstances(ctx EvalContext, addr addrs.ActionInstance, callRange *hcl.Range) (addrs.Map[addrs.ActionInstance, cty.Value], tfdiags.Diagnostics) {
	all := addrs.MakeMap[addrs.ActionInstance, cty.Value]()

	switch key := addr.Key.(type) {
	case addrs.IntKey, addrs.StringKey:
		val, diags := n.EvalInstance(ctx, key, callRange)
		if diags.HasErrors() {
			return all, diags
		}
		all.Put(addr, val)
		return all, diags

	default:
		return n.eval(ctx, addr.Key, callRange)
	}
}

// EvalInstance returns the value from the expanded action block
func (n *NodeActionConfig) EvalInstance(ctx EvalContext, key addrs.InstanceKey, callRange *hcl.Range) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
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

	vals, diags := n.eval(ctx, key, callRange)
	if diags.HasErrors() {
		return cty.DynamicVal, diags
	}

	val := vals.Get(n.Addr.Action.Instance(key))
	return val, diags
}

// Eval one or more instances of the action. This function expects that the key
// is already validated for the the calling context, and will not produce
// diagnostics for incorrect key types.
func (n *NodeActionConfig) eval(ctx EvalContext, key addrs.InstanceKey, callRange *hcl.Range) (addrs.Map[addrs.ActionInstance, cty.Value], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	all := addrs.MakeMap[addrs.ActionInstance, cty.Value]()

	actionInstances, diags := n.repetitionData(ctx)
	if diags.HasErrors() {
		return all, diags
	}

	switch key := key.(type) {
	case addrs.IntKey:
		if int(key) >= len(actionInstances) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reference to non-existent action instance",
				Detail:   fmt.Sprintf("The given key %d does not identify an instance of %s", key, n.Addr.Absolute(ctx.Path())),
				Subject:  callRange,
			})
			return all, diags
		}

		val, evalDiags := n.evalInstance(ctx, actionInstances[int(key)])
		diags = append(diags, evalDiags...)
		all.Put(n.Addr.Action.Instance(key), val)

		return all, diags

	case addrs.StringKey:
		for _, inst := range actionInstances {
			// find the one instance we're looking for
			if inst.EachKey.AsString() != string(key) {
				continue
			}
			val, evalDiags := n.evalInstance(ctx, inst)
			diags = diags.Append(evalDiags)
			if evalDiags.HasErrors() {
				return all, diags
			}

			all.Put(n.Addr.Action.Instance(key), val)
		}

		if all.Len() == 0 {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reference to non-existent action instance",
				Detail:   fmt.Sprintf("The given key %s does not identify an instance of %s", key, n.Addr.Absolute(ctx.Path())),
				Subject:  callRange,
			})
			return all, diags
		}

		return all, diags

	default:
		for _, inst := range actionInstances {
			val, evalDiags := n.evalInstance(ctx, inst)
			diags = diags.Append(evalDiags)
			if diags.HasErrors() {
				return all, diags
			}

			// we need to generate keys out of any new repetition Data
			switch {
			case inst.CountIndex != cty.NilVal:
				idx, _ := inst.CountIndex.AsBigFloat().Int64()
				key = addrs.IntKey(int(idx))
			case inst.EachKey != cty.NilVal:
				key = addrs.StringKey(inst.EachKey.AsString())
			}

			all.Put(n.Addr.Action.Instance(key), val)
		}

		return all, diags
	}
}

func (n *NodeActionConfig) evalInstance(ctx EvalContext, repData instances.RepetitionData) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// This should have been caught already
	if n.Schema == nil {
		panic("action eval called without a schema")
	}

	configVal := cty.NullVal(n.Schema.ConfigSchema.ImpliedType())
	if n.Config.Config != nil {
		var configDiags tfdiags.Diagnostics
		configVal, _, configDiags = ctx.EvaluateBlock(n.Config.Config, n.Schema.ConfigSchema.DeepCopy(), nil, repData)
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

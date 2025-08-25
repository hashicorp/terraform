// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// GraphNodeConfigAction is implemented by any nodes that represent an action.
// The type of operation cannot be assumed, only that this node represents
// the given resource.
type GraphNodeConfigAction interface {
	ActionAddr() addrs.ConfigAction
}

// nodeExpandActionDeclaration represents an action config block in a configuration module,
// which has not yet been expanded.
type nodeExpandActionDeclaration struct {
	Addr   addrs.ConfigAction
	Config configs.Action

	Schema           *providers.ActionSchema
	ResolvedProvider addrs.AbsProviderConfig
}

var (
	_ GraphNodeConfigAction       = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeReferenceable      = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeReferencer         = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeDynamicExpandable  = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeProviderConsumer   = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeAttachActionSchema = (*nodeExpandActionDeclaration)(nil)
)

// nodeExpandActionDeclaration's Execute function handles action configuration
// validation.
func (n *nodeExpandActionDeclaration) Execute(ctx EvalContext) tfdiags.Diagnostics {
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

	configVal, _, valDiags := ctx.EvaluateBlock(n.Config.Config, schema.ConfigSchema, nil, keyData)
	diags = diags.Append(valDiags)
	if valDiags.HasErrors() {
		return diags
	}

	// Use unmarked value for validate request
	unmarkedConfigVal, _ := configVal.UnmarkDeep()
	log.Printf("[TRACE] Validating config for %q", n.Addr)
	req := providers.ValidateActionConfigRequest{
		TypeName: n.Config.Type,
		Config:   unmarkedConfigVal,
		// LinkedResources: lifecycle actions not yet implemented
	}

	resp := provider.ValidateActionConfig(req)
	diags = diags.Append(resp.Diagnostics.InConfigBody(n.Config.Config, n.Addr.String()))

	return diags
}

func (n *nodeExpandActionDeclaration) Name() string {
	return n.Addr.String() + " (expand)"
}

func (n *nodeExpandActionDeclaration) ActionAddr() addrs.ConfigAction {
	return n.Addr
}

func (n *nodeExpandActionDeclaration) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr.Action}
}

// GraphNodeModulePath
func (n *nodeExpandActionDeclaration) ModulePath() addrs.Module {
	return n.Addr.Module
}

// GraphNodeAttachActionSchema
func (n *nodeExpandActionDeclaration) AttachActionSchema(schema *providers.ActionSchema) {
	n.Schema = schema
}

func (n *nodeExpandActionDeclaration) DotNode(string, *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: n.Name(),
	}
}

// GraphNodeReferencer
func (n *nodeExpandActionDeclaration) References() []*addrs.Reference {
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

func (n *nodeExpandActionDeclaration) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var g Graph
	var diags tfdiags.Diagnostics
	expander := ctx.InstanceExpander()
	moduleInstances := expander.ExpandModule(n.Addr.Module, false)

	for _, module := range moduleInstances {
		absActAddr := n.Addr.Absolute(module)

		// Check if the actions language experiment is enabled for this module.
		moduleCtx := evalContextForModuleInstance(ctx, module)

		// recordActionData is responsible for informing the expander of what
		// repetition mode this resource has, which allows expander.ExpandResource
		// to work below.
		moreDiags := n.recordActionData(moduleCtx, absActAddr)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return nil, diags
		}

		// Expand the action instances for this module.
		for _, absActInstance := range expander.ExpandAction(absActAddr) {
			node := NodeActionDeclarationInstance{
				Addr:             absActInstance,
				Config:           n.Config,
				Schema:           n.Schema,
				ResolvedProvider: n.ResolvedProvider,
			}

			g.Add(&node)
		}
	}

	addRootNodeToGraph(&g)

	return &g, diags
}

func (n *nodeExpandActionDeclaration) recordActionData(ctx EvalContext, addr addrs.AbsAction) (diags tfdiags.Diagnostics) {

	// We'll record our expansion decision in the shared "expander" object
	// so that later operations (i.e. DynamicExpand and expression evaluation)
	// can refer to it. Since this node represents the abstract module, we need
	// to expand the module here to create all resources.
	expander := ctx.InstanceExpander()

	// Allowing unknown values in count and for_each is a top-level plan option.
	//
	// If this is false then the codepaths that handle unknown values below
	// become unreachable, because the evaluate functions will reject unknown
	// values as an error.
	// allowUnknown := ctx.Deferrals().DeferralAllowed()
	allowUnknown := false

	switch {
	case n.Config.Count != nil:
		count, countDiags := evaluateCountExpression(n.Config.Count, ctx, allowUnknown)
		diags = diags.Append(countDiags)
		if countDiags.HasErrors() {
			return diags
		}

		if count >= 0 {
			expander.SetActionCount(addr.Module, n.Addr.Action, count)
		} else {
			// -1 represents "unknown"
			expander.SetActionCountUnknown(addr.Module, n.Addr.Action)
		}

	case n.Config.ForEach != nil:
		forEach, known, forEachDiags := evaluateForEachExpression(n.Config.ForEach, ctx, allowUnknown)
		diags = diags.Append(forEachDiags)
		if forEachDiags.HasErrors() {
			return diags
		}

		// This method takes care of all of the business logic of updating this
		// while ensuring that any existing instances are preserved, etc.
		if known {
			expander.SetActionForEach(addr.Module, n.Addr.Action, forEach)
		} else {
			expander.SetActionForEachUnknown(addr.Module, n.Addr.Action)
		}

	default:
		expander.SetActionSingle(addr.Module, n.Addr.Action)
	}

	return diags
}

// GraphNodeProviderConsumer
func (n *nodeExpandActionDeclaration) ProvidedBy() (addrs.ProviderConfig, bool) {
	// Once the provider is fully resolved, we can return the known value.
	if n.ResolvedProvider.Provider.Type != "" {
		return n.ResolvedProvider, true
	}

	// Since we always have a config, we can use it
	relAddr := n.Config.ProviderConfigAddr()
	return addrs.LocalProviderConfig{
		LocalName: relAddr.LocalName,
		Alias:     relAddr.Alias,
	}, false
}

// GraphNodeProviderConsumer
func (n *nodeExpandActionDeclaration) Provider() addrs.Provider {
	return n.Config.Provider
}

// GraphNodeProviderConsumer
func (n *nodeExpandActionDeclaration) SetProvider(p addrs.AbsProviderConfig) {
	n.ResolvedProvider = p
}

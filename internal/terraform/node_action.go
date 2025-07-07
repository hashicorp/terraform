// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/experiments"
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
	_ GraphNodeConfigAction      = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeReferenceable     = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeReferencer        = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeDynamicExpandable = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeProviderConsumer  = (*nodeExpandActionDeclaration)(nil)
)

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

// GraphNodeAttachActionSchema impl
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
		allowActions := moduleCtx.LanguageExperimentActive(experiments.Actions)
		if !allowActions {
			summary := fmt.Sprintf("Actions experiment not enabled for module %s", module)
			if module.IsRoot() {
				summary = "Actions experiment not enabled"
			}
			return nil, diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				summary,
				"The actions experiment must be enabled in order to use actions in your configuration. You can enable it by adding a terraform { experiments = [actions] } block to your configuration.",
			))
		}

		// The rest of our work here needs to know which module instance it's
		// working in, so that it can evaluate expressions in the appropriate scope.
		_, err := n.expandActionInstances(ctx, absActAddr, &g)
		diags = diags.Append(err)
	}

	return &g, diags
}

func (n *nodeExpandActionDeclaration) expandActionInstances(globalCtx EvalContext, actAddr addrs.AbsAction, g *Graph) ([]addrs.AbsActionInstance, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// The rest of our work here needs to know which module instance it's
	// working in, so that it can evaluate expressions in the appropriate scope.
	moduleCtx := evalContextForModuleInstance(globalCtx, actAddr.Module)

	// writeResourceState is responsible for informing the expander of what
	// repetition mode this resource has, which allows expander.ExpandResource
	// to work below.
	moreDiags := n.recordActionData(moduleCtx, actAddr)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	// Before we expand our resource into potentially many resource instances,
	// we'll verify that any mention of this resource in n.forceReplace is
	// consistent with the repetition mode of the resource. In other words,
	// we're aiming to catch a situation where naming a particular resource
	// instance would require an instance key but the given address has none.
	expander := moduleCtx.InstanceExpander()
	instanceAddrs := expander.ExpandAction(actAddr)

	// NOTE: The actual interpretation of n.forceReplace to produce replace
	// actions is in the per-instance function we're about to call, because
	// we need to evaluate it on a per-instance basis.

	// Our graph builder mechanism expects to always be constructing new
	// graphs rather than adding to existing ones, so we'll first
	// construct a subgraph just for this individual modules's instances and
	// then we'll steal all of its nodes and edges to incorporate into our
	// main graph which contains all of the resource instances together.
	instG, instDiags := n.actionInstanceSubgraph(actAddr, instanceAddrs)
	if instDiags.HasErrors() {
		diags = diags.Append(instDiags)
		return nil, diags
	}
	g.Subsume(&instG.AcyclicGraph.Graph)

	return instanceAddrs, diags
}

func (n *nodeExpandActionDeclaration) actionInstanceSubgraph(addr addrs.AbsAction, instanceAddrs []addrs.AbsActionInstance) (*Graph, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Start creating the steps
	steps := []GraphTransformer{
		// Expand the count or for_each (if present)
		&ActionCountTransformer{
			Schema:           n.Schema,
			Config:           n.Config,
			Addr:             n.Addr,
			InstanceAddrs:    instanceAddrs,
			ResolvedProvider: n.ResolvedProvider,
		},

		// Connect references so ordering is correct
		&ReferenceTransformer{},

		// Make sure there is a single root
		&RootTransformer{},
	}

	// Build the graph
	b := &BasicGraphBuilder{
		Steps: steps,
		Name:  "nodeExpandActionDeclaration",
	}
	graph, graphDiags := b.Build(addr.Module)
	diags = diags.Append(graphDiags)

	return graph, diags
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

	return addrs.AbsProviderConfig{
		Provider: n.Provider(),
		Module:   addrs.RootModule, // TODO: Deal with modules
	}, false
}

// GraphNodeProviderConsumer
func (n *nodeExpandActionDeclaration) Provider() addrs.Provider {
	// TODO: Handle provider field
	return addrs.ImpliedProviderForUnqualifiedType(n.Addr.Action.ImpliedProvider())
}

// GraphNodeProviderConsumer
func (n *nodeExpandActionDeclaration) SetProvider(p addrs.AbsProviderConfig) {
	n.ResolvedProvider = p
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
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
	*NodeAbstractAction
}

var (
	_ GraphNodeConfigAction       = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeReferenceable      = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeReferencer         = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeDynamicExpandable  = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeProviderConsumer   = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeAttachActionSchema = (*nodeExpandActionDeclaration)(nil)
)

func (n *nodeExpandActionDeclaration) Name() string {
	return n.Addr.String() + " (expand)"
}

func (n *nodeExpandActionDeclaration) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var g Graph
	var diags tfdiags.Diagnostics
	expander := ctx.InstanceExpander()
	moduleInstances := expander.ExpandModule(n.Addr.Module, false)

	// The possibility of partial-expanded modules and resources is guarded by a
	// top-level option for the whole plan, so that we can preserve mainline
	// behavior for the modules runtime. So, we currently branch off into an
	// entirely-separate codepath in those situations, at the expense of
	// duplicating some of the logic for behavior this method would normally
	// handle.
	if ctx.Deferrals().DeferralAllowed() {
		pem := expander.UnknownModuleInstances(n.Addr.Module, false)

		for _, moduleAddr := range pem {
			resourceAddr := moduleAddr.Action(n.Addr.Action)

			// And add a node to the graph for this action.
			g.Add(&NodeActionDeclarationPartialExpanded{
				addr:             resourceAddr,
				config:           n.Config,
				resolvedProvider: n.ResolvedProvider,
			})
		}
		addRootNodeToGraph(&g)
	}

	for _, module := range moduleInstances {
		absActAddr := n.Addr.Absolute(module)

		// Check if the actions language experiment is enabled for this module.
		moduleCtx := evalContextForModuleInstance(ctx, module)

		// recordActionData is responsible for informing the expander of what
		// repetition mode this resource has, which allows expander.ExpandAction
		// to work below.
		moreDiags := n.recordActionData(moduleCtx, absActAddr)

		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return nil, diags
		}

		_, knownInstKeys, haveUnknownKeys := expander.ActionInstanceKeys(absActAddr)
		if haveUnknownKeys {
			node := NodeActionDeclarationInstance{
				Addr:             absActAddr.Instance(addrs.WildcardKey),
				Config:           n.Config,
				Schema:           n.Schema,
				ResolvedProvider: n.ResolvedProvider,
				Dependencies:     n.Dependencies,
			}
			g.Add(&node)
		} else {
			// Expand the action instances for this module.
			for _, knownInstKey := range knownInstKeys {
				node := NodeActionDeclarationInstance{
					Addr:             absActAddr.Instance(knownInstKey),
					Config:           n.Config,
					Schema:           n.Schema,
					ResolvedProvider: n.ResolvedProvider,
					Dependencies:     n.Dependencies,
				}

				g.Add(&node)
			}
		}

		addRootNodeToGraph(&g)
	}

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
	allowUnknown := ctx.Deferrals().DeferralAllowed()

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

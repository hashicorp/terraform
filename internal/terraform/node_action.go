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
	_ GraphNodeDynamicExpandable = (*nodeExpandActionDeclaration)(nil)
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
			actionAddr := moduleAddr.Action(n.Addr.Action)

			// And add a node to the graph for this action.
			g.Add(&NodeActionDeclarationPartialExpanded{
				addr:             actionAddr,
				config:           n.Config,
				Schema:           n.Schema,
				resolvedProvider: n.ResolvedProvider,
			})
		}
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
			// this should never happen, n.recordActionData explicitly sets
			// allowUnknown to be false, so we should pick up diagnostics
			// during that call instance reaching this branch.
			panic("found unknown keys in action instance")
		}

		// Expand the action instances for this module.
		for _, knownInstKey := range knownInstKeys {
			node := NodeActionDeclarationInstance{
				Addr:             absActAddr.Instance(knownInstKey),
				Config:           &n.Config,
				Schema:           n.Schema,
				ResolvedProvider: n.ResolvedProvider,
				Dependencies:     n.Dependencies,
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

	// For now, action instances cannot evaluate to unknown. When an action
	// would have an unknown instance key, we'd want to defer executing that
	// action, and in turn defer executing the triggering resource. Delayed
	// deferrals are not currently possible (we need to reconfigure exactly how
	// deferrals are checked) so for now deferred actions are simply blocked.

	switch {
	case n.Config.Count != nil:
		count, countDiags := evaluateCountExpression(n.Config.Count, ctx, false)
		diags = diags.Append(countDiags)
		if countDiags.HasErrors() {
			return diags
		}

		if count >= 0 {
			expander.SetActionCount(addr.Module, n.Addr.Action, count)
		} else {
			// this should not be possible as allowUnknown was set to false
			// in the evaluateCountExpression function call.
			panic("evaluateCountExpression returned unknown")
		}

	case n.Config.ForEach != nil:
		forEach, known, forEachDiags := evaluateForEachExpression(n.Config.ForEach, ctx, false)
		diags = diags.Append(forEachDiags)
		if forEachDiags.HasErrors() {
			return diags
		}

		// This method takes care of all of the business logic of updating this
		// while ensuring that any existing instances are preserved, etc.
		if known {
			expander.SetActionForEach(addr.Module, n.Addr.Action, forEach)
		} else {
			// this should not be possible as allowUnknown was set to false
			// in the evaluateForEachExpression function call.
			panic("evaluateForEachExpression returned unknown")
		}

	default:
		expander.SetActionSingle(addr.Module, n.Addr.Action)
	}

	return diags
}

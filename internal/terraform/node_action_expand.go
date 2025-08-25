// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodeExpandActionDeclaration represents an action config block in a configuration module,
// which has not yet been expanded.
type nodeExpandActionDeclaration struct {
	*NodeAbstractActionDeclaration
}

var (
	_ GraphNodeConfigAction       = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeReferenceable      = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeReferencer         = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeDynamicExpandable  = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeProviderConsumer   = (*nodeExpandActionDeclaration)(nil)
	_ GraphNodeAttachActionSchema = (*nodeExpandActionDeclaration)(nil)
)

func (n *nodeExpandActionDeclaration) String() string {
	return fmt.Sprintf("%s (expand)", n.Addr)
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

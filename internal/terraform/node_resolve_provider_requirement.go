// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeResolveProviderRequirements struct {
	Addr   addrs.ModuleInstance
	Module *configs.Module
	Exprs  map[string]*configs.ProviderRequirementExpr
}

var (
	_ GraphNodeExecutable     = (*nodeResolveProviderRequirements)(nil)
	_ GraphNodeReferencer     = (*nodeResolveProviderRequirements)(nil)
	_ GraphNodeModuleInstance = (*nodeResolveProviderRequirements)(nil)
	_ dag.NamedVertex         = (*nodeResolveProviderRequirements)(nil)
)

func (n *nodeResolveProviderRequirements) Name() string {
	return n.Addr.String()
}

func (n *nodeResolveProviderRequirements) Execute(
	ctx EvalContext,
	_ walkOperation,
) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	for name, expr := range n.Exprs {
		rp, rpDiags := n.resolveProvider(name, expr, ctx)
		diags = append(diags, rpDiags...)
		if rpDiags.HasErrors() {
			continue
		}

		n.Module.ProviderRequirements.RequiredProviders[name] = rp
		if n.Module.StateStore != nil && n.Module.StateStore.Provider != nil && n.Module.StateStore.Provider.Name == name {
			n.Module.StateStore.ProviderAddr = rp.Type
		}
	}

	n.Module.GatherProviderLocalNames()

	return diags
}

func (n *nodeResolveProviderRequirements) resolveProvider(
	name string,
	expr *configs.ProviderRequirementExpr,
	ctx EvalContext,
) (*configs.RequiredProvider, tfdiags.Diagnostics) {
	return ResolveProvider(name, expr, ctx)
}

func (n *nodeResolveProviderRequirements) References() []*addrs.Reference {
	var refs []*addrs.Reference
	for _, expr := range n.Exprs {
		if expr.SourceExpr != nil {
			sourceRefs, _ := langrefs.ReferencesInExpr(
				addrs.ParseRef, expr.SourceExpr,
			)
			refs = append(refs, sourceRefs...)
		}

		if expr.VersionExpr != nil {
			versionRefs, _ := langrefs.ReferencesInExpr(
				addrs.ParseRef, expr.VersionExpr,
			)
			refs = append(refs, versionRefs...)
		}
	}

	return refs
}

func (n *nodeResolveProviderRequirements) Path() addrs.ModuleInstance {
	return n.Addr
}

func (n *nodeResolveProviderRequirements) ModulePath() addrs.Module {
	return n.Addr.Module()
}

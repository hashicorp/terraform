// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NodeAbstractActionInstance represents an action in a particular module.
//
// Action configuration blocks don't do anything by themselves, they are just
// coming into effect when they are triggered. We expand them here so that
// when they are referenced we can get the configuration for the action directly.
type NodeAbstractActionInstance struct {
	Addr             addrs.AbsActionInstance
	Config           *configs.Action
	Schema           *providers.ActionSchema
	ResolvedProvider addrs.AbsProviderConfig
	Dependencies     []addrs.ConfigResource
}

var (
	_ GraphNodeModuleInstance = (*NodeAbstractActionInstance)(nil)
	_ GraphNodeExecutable     = (*NodeAbstractActionInstance)(nil)
	_ GraphNodeReferencer     = (*NodeAbstractActionInstance)(nil)
	_ GraphNodeReferenceable  = (*NodeAbstractActionInstance)(nil)
)

func (n *NodeAbstractActionInstance) Name() string {
	return n.Addr.String()
}

func (n *NodeAbstractActionInstance) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

func (n *NodeAbstractActionInstance) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
	deferrals := ctx.Deferrals()
	if deferrals.DeferralAllowed() && deferrals.ShouldDeferAction(n.Dependencies) {
		deferrals.ReportActionDeferred(n.Addr, providers.DeferredReasonDeferredPrereq)
	}
	return nil
}

// GraphNodeReferenceable
func (n *NodeAbstractActionInstance) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr.Action, n.Addr.Action.Action}
}

// GraphNodeReferencer
func (n *NodeAbstractActionInstance) References() []*addrs.Reference {
	var result []*addrs.Reference
	c := n.Config
	countRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, c.Count)
	result = append(result, countRefs...)
	forEachRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, c.ForEach)
	result = append(result, forEachRefs...)

	if n.Schema != nil {
		configRefs, _ := langrefs.ReferencesInBlock(addrs.ParseRef, c.Config, n.Schema.ConfigSchema)
		result = append(result, configRefs...)
	}

	return result
}

func (n *NodeAbstractActionInstance) ModulePath() addrs.Module {
	return n.Addr.Module.Module()
}

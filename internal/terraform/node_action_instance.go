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
	var diags tfdiags.Diagnostics

	deferrals := ctx.Deferrals()
	if deferrals.DeferralAllowed() && deferrals.ShouldDeferAction(n.Dependencies) {
		deferrals.ReportActionDeferred(n.Addr, providers.DeferredReasonDeferredPrereq)
		return diags
	}

	// This should have been caught already
	if n.Schema == nil {
		panic("NodeActionDeclarationInstance.Execute called without a schema")
	}

	allInsts := ctx.InstanceExpander()
	keyData := allInsts.GetActionInstanceRepetitionData(n.Addr)

	if n.Config.Config != nil {
		var configDiags tfdiags.Diagnostics
		configVal, _, configDiags := ctx.EvaluateBlock(n.Config.Config, n.Schema.ConfigSchema.DeepCopy(), nil, keyData)

		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			return diags
		}

		valDiags := validateResourceForbiddenEphemeralValues(ctx, configVal, n.Schema.ConfigSchema)
		diags = diags.Append(valDiags.InConfigBody(n.Config.Config, n.Addr.String()))

		var deprecationDiags tfdiags.Diagnostics
		_, deprecationDiags = ctx.Deprecations().ValidateAndUnmarkConfig(configVal, n.Schema.ConfigSchema, n.ModulePath())
		diags = diags.Append(deprecationDiags.InConfigBody(n.Config.Config, n.Addr.String()))

		if diags.HasErrors() {
			return diags
		}
	}

	return diags
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

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NodeActionDeclarationInstance represents an action in a particular module.
//
// Action Declarations don't do anything by themselves, they are just
// coming into effect when they are triggered. We expand them here so that
// when they are referenced we can get the configuration for the action directly.
type NodeActionDeclarationInstance struct {
	Addr             addrs.AbsActionInstance
	Config           *configs.Action
	Schema           *providers.ActionSchema
	ResolvedProvider addrs.AbsProviderConfig
	Dependencies     []addrs.ConfigResource
}

var (
	_ GraphNodeModuleInstance = (*NodeActionDeclarationInstance)(nil)
	_ GraphNodeExecutable     = (*NodeActionDeclarationInstance)(nil)
	_ GraphNodeReferencer     = (*NodeActionDeclarationInstance)(nil)
	_ GraphNodeReferenceable  = (*NodeActionDeclarationInstance)(nil)
)

func (n *NodeActionDeclarationInstance) Name() string {
	return n.Addr.String()
}

func (n *NodeActionDeclarationInstance) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

func (n *NodeActionDeclarationInstance) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
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

	configVal := cty.NullVal(n.Schema.ConfigSchema.ImpliedType())
	if n.Config.Config != nil {
		var configDiags tfdiags.Diagnostics
		configVal, _, configDiags = ctx.EvaluateBlock(n.Config.Config, n.Schema.ConfigSchema.DeepCopy(), nil, keyData)

		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			return diags
		}

		valDiags := validateResourceForbiddenEphemeralValues(ctx, configVal, n.Schema.ConfigSchema)
		diags = diags.Append(valDiags.InConfigBody(n.Config.Config, n.Addr.String()))

		var deprecationDiags tfdiags.Diagnostics
		configVal, deprecationDiags = ctx.Deprecations().ValidateConfig(configVal, n.Schema.ConfigSchema, n.ModulePath())
		diags = diags.Append(deprecationDiags.InConfigBody(n.Config.Config, n.Addr.String()))

		if diags.HasErrors() {
			return diags
		}
	}

	ctx.Actions().AddActionInstance(n.Addr, configVal, n.ResolvedProvider)
	return diags
}

// GraphNodeReferenceable
func (n *NodeActionDeclarationInstance) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr.Action, n.Addr.Action.Action}
}

// GraphNodeReferencer
func (n *NodeActionDeclarationInstance) References() []*addrs.Reference {
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

func (n *NodeActionDeclarationInstance) ModulePath() addrs.Module {
	return n.Addr.Module.Module()
}

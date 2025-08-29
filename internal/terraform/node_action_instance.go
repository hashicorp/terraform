// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// NodeActionDeclarationInstance represents an action in a particular module.
//
// Action Declarations don't do anything by themselves, they are just
// coming into effect when they are triggered. We expand them here so that
// when they are referenced we can get the configuration for the action directly.
type NodeActionDeclarationInstance struct {
	Addr             addrs.AbsActionInstance
	Config           configs.Action
	Schema           *providers.ActionSchema
	ResolvedProvider addrs.AbsProviderConfig
}

var (
	_ GraphNodeModuleInstance = (*NodeActionDeclarationInstance)(nil)
	_ GraphNodeExecutable     = (*NodeActionDeclarationInstance)(nil)
	_ GraphNodeReferencer     = (*NodeActionDeclarationInstance)(nil)
	_ GraphNodeReferenceable  = (*NodeActionDeclarationInstance)(nil)
	_ dag.GraphNodeDotter     = (*NodeActionDeclarationInstance)(nil)
)

func (n *NodeActionDeclarationInstance) Name() string {
	return n.Addr.String()
}

func (n *NodeActionDeclarationInstance) DotNode(string, *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: n.Name(),
	}
}

func (n *NodeActionDeclarationInstance) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

func (n *NodeActionDeclarationInstance) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if n.Addr.Action.Key == addrs.WildcardKey {
		if ctx.Deferrals().DeferralAllowed() {
			ctx.Deferrals().ReportActionDeferred(n.Addr, providers.DeferredReasonInstanceCountUnknown)
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Action expansion was deferred",
				Detail:   "Deferral is not allowed in this context",
				Subject:  n.Config.DeclRange.Ptr(),
			})
		}
		return diags
	}

	// This should have been caught already
	if n.Schema == nil {
		panic("NodeActionDeclarationInstance.Execute called without a schema")
	}

	// We currently only support unlinked actions, so we send a diagnostic for other types
	if n.Schema.Lifecycle != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Lifecycle actions are not supported",
			Detail:   "This version of Terraform does not support lifecycle actions",
			Subject:  n.Config.DeclRange.Ptr(),
		})
		return diags
	}

	if n.Schema.Linked != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Linked actions are not supported",
			Detail:   "This version of Terraform does not support linked actions",
			Subject:  n.Config.DeclRange.Ptr(),
		})
		return diags
	}

	allInsts := ctx.InstanceExpander()
	keyData := allInsts.GetActionInstanceRepetitionData(n.Addr)

	configVal := cty.NullVal(n.Schema.ConfigSchema.ImpliedType())
	if n.Config.Config != nil {
		var configDiags tfdiags.Diagnostics
		configVal, _, configDiags = ctx.EvaluateBlock(n.Config.Config, n.Schema.ConfigSchema.DeepCopy(), nil, keyData)

		diags = diags.Append(configDiags)
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

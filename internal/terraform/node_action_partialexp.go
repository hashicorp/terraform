// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// NodeActionDeclarationPartialExpanded is a graph node that stands in for
// an unbounded set of potential action instances that we don't yet know.
//
// Its job is to check the configuration as much as we can with the information
// that's available (so we can raise an error early if something is clearly
// wrong across _all_ potential instances) and to record a placeholder value
// for use when evaluating other objects that refer to this resource.
//
// This is the partial-expanded equivalent of NodeActionDeclarationInstance.
type NodeActionDeclarationPartialExpanded struct {
	addr             addrs.PartialExpandedAction
	config           configs.Action
	Schema           *providers.ActionSchema
	resolvedProvider addrs.AbsProviderConfig
}

var (
	_ graphNodeEvalContextScope = (*NodeActionDeclarationPartialExpanded)(nil)
	_ GraphNodeExecutable       = (*NodeActionDeclarationPartialExpanded)(nil)
)

// Name implements [dag.NamedVertex].
func (n *NodeActionDeclarationPartialExpanded) Name() string {
	return n.addr.String()
}

// Path implements graphNodeEvalContextScope.
func (n *NodeActionDeclarationPartialExpanded) Path() evalContextScope {
	if moduleAddr, ok := n.addr.ModuleInstance(); ok {
		return evalContextModuleInstance{Addr: moduleAddr}
	} else if moduleAddr, ok := n.addr.PartialExpandedModule(); ok {
		return evalContextPartialExpandedModule{Addr: moduleAddr}
	} else {
		// Should not get here: at least one of the two cases above
		// should always be true for any valid addrs.PartialExpandedResource
		panic("addrs.PartialExpandedResource has neither a partial-expanded or a fully-expanded module instance address")
	}
}

func (n *NodeActionDeclarationPartialExpanded) ActionAddr() addrs.ConfigAction {
	return n.addr.ConfigAction()
}

// Execute implements GraphNodeExecutable.
func (n *NodeActionDeclarationPartialExpanded) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	ctx.Deferrals().ReportActionExpansionDeferred(n.addr)
	configVal := cty.NullVal(n.Schema.ConfigSchema.ImpliedType())
	if n.config.Config != nil {
		var configDiags tfdiags.Diagnostics
		configVal, _, configDiags = ctx.EvaluateBlock(n.config.Config, n.Schema.ConfigSchema.DeepCopy(), nil, instances.TotallyUnknownRepetitionData)

		diags = diags.Append(configDiags)
		if diags.HasErrors() {
			return diags
		}
		var deprecationDiags tfdiags.Diagnostics
		configVal, deprecationDiags = ctx.Deprecations().ValidateConfig(configVal, n.Schema.ConfigSchema, n.ActionAddr().Module)
		diags = diags.Append(deprecationDiags.InConfigBody(n.config.Config, n.ActionAddr().String()))
		if diags.HasErrors() {
			return diags
		}
	}
	ctx.Actions().AddPartialExpandedAction(n.addr, configVal, n.resolvedProvider)
	return nil
}

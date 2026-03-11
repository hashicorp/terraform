// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type TestInputValue struct {
	Value *terraform.InputValue
}

var (
	_ GraphNodeExecutable    = (*NodeVariableDefinition)(nil)
	_ GraphNodeReferenceable = (*NodeVariableDefinition)(nil)
)

type NodeVariableDefinition struct {
	Address string
	Config  *configs.Variable
	File    *moduletest.File
}

func (n *NodeVariableDefinition) Name() string {
	return fmt.Sprintf("var.%s", n.Address)
}

func (n *NodeVariableDefinition) Referenceable() addrs.Referenceable {
	return &addrs.InputVariable{Name: n.Address}
}

func (n *NodeVariableDefinition) Execute(ctx *EvalContext) {
	if ctx.Stopped() || ctx.Cancelled() {
		return // don't evaluate anything when stopped or cancelled
	}

	input, diags := ctx.EvaluateUnparsedVariable(n.Address, n.Config)
	if input != nil {
		n.File.AppendDiagnostics(diags)
		if diags.HasErrors() {
			ctx.SetVariableStatus(n.Address, moduletest.Error)
			return
		}
	} else {
		input = &terraform.InputValue{
			Value: cty.NilVal,
		}
	}

	value, diags := terraform.PrepareFinalInputVariableValue(addrs.AbsInputVariableInstance{
		Module: addrs.RootModuleInstance,
		Variable: addrs.InputVariable{
			Name: n.Address,
		},
	}, input, n.Config)
	n.File.AppendDiagnostics(diags)
	if diags.HasErrors() {
		ctx.SetVariableStatus(n.Address, moduletest.Error)
		return
	}

	ctx.SetVariable(n.Address, &terraform.InputValue{
		Value:       value,
		SourceType:  terraform.ValueFromConfig,
		SourceRange: tfdiags.SourceRangeFromHCL(n.Config.DeclRange),
	})
}

var (
	_ GraphNodeExecutable    = (*NodeVariableExpression)(nil)
	_ GraphNodeReferencer    = (*NodeVariableExpression)(nil)
	_ GraphNodeReferenceable = (*NodeVariableExpression)(nil)
)

type NodeVariableExpression struct {
	Address string
	Expr    hcl.Expression
	File    *moduletest.File
}

func (n *NodeVariableExpression) Name() string {
	return fmt.Sprintf("var.%s", n.Address)
}

func (n *NodeVariableExpression) Referenceable() addrs.Referenceable {
	return &addrs.InputVariable{Name: n.Address}
}

func (n *NodeVariableExpression) References() []*addrs.Reference {
	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, n.Expr)
	return refs
}

func (n *NodeVariableExpression) Execute(ctx *EvalContext) {
	refs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, n.Expr)

	if !ctx.ReferencesCompleted(refs) {

		// before we do anything with the diags, we're going to check that all
		// of the references we know about were successfully parsed. If not,
		// then we'll skip actually assessing this node.

		ctx.SetVariableStatus(n.Address, moduletest.Skip)
		return
	}

	if ctx.Stopped() || ctx.Cancelled() {
		return // don't evaluate anything when stopped or cancelled
	}

	n.File.AppendDiagnostics(refDiags)
	if refDiags.HasErrors() {
		ctx.SetVariableStatus(n.Address, moduletest.Error)
		return
	}

	evalContext, moreDiags := ctx.HclContext(refs)
	n.File.AppendDiagnostics(moreDiags)
	if moreDiags.HasErrors() {
		ctx.SetVariableStatus(n.Address, moduletest.Error)
		return
	}

	var diags tfdiags.Diagnostics
	value, valueDiags := n.Expr.Value(evalContext)
	n.File.AppendDiagnostics(diags.Append(valueDiags))
	if valueDiags.HasErrors() {
		ctx.SetVariableStatus(n.Address, moduletest.Error)
		return
	}

	ctx.SetVariable(n.Address, &terraform.InputValue{
		Value:       value,
		SourceType:  terraform.ValueFromConfig,
		SourceRange: tfdiags.SourceRangeFromHCL(n.Expr.Range()),
	})
}

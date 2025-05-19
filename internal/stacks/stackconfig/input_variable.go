// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/typeexpr"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// InputVariable is a declaration of an input variable within a stack
// configuration. Callers must provide the values for these variables.
type InputVariable struct {
	Name string

	Type         TypeConstraint
	DefaultValue cty.Value
	Description  string

	Sensitive bool
	Ephemeral bool

	DeclRange tfdiags.SourceRange
}

// TypeConstraint represents all of the type constraint information for either
// an input variable or an output value.
//
// After initial decoding only Expression is populated, and it has not yet been
// analyzed at all so is not even guaranteed to be a valid type constraint
// expression.
//
// For configurations loaded through the main entry point [LoadConfigDir],
// Constraint is populated with the result of decoding Expression as a type
// constraint only if the expression is a valid type constraint expression.
// When loading through shallower entry points such as [DecodeFileBody],
// Constraint is not populated.
//
// Defaults is populated only if Constraint is, and if not nil represents any
// default values from the type constraint expression.
type TypeConstraint struct {
	Expression hcl.Expression
	Constraint cty.Type
	Defaults   *typeexpr.Defaults
}

func decodeInputVariableBlock(block *hcl.Block) (*InputVariable, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &InputVariable{
		Name:      block.Labels[0],
		DeclRange: tfdiags.SourceRangeFromHCL(block.DefRange),
	}
	if !hclsyntax.ValidIdentifier(ret.Name) {
		diags = diags.Append(invalidNameDiagnostic(
			"Invalid name for input variable",
			block.LabelRanges[0],
		))
		return nil, diags
	}

	content, hclDiags := block.Body.Content(inputVariableBlockSchema)
	diags = diags.Append(hclDiags)

	if attr, ok := content.Attributes["type"]; ok {
		ret.Type.Expression = attr.Expr
	}
	if attr, ok := content.Attributes["default"]; ok {
		val, hclDiags := attr.Expr.Value(nil)
		diags = diags.Append(hclDiags)
		if val == cty.NilVal {
			val = cty.DynamicVal
		}
		ret.DefaultValue = val
	}
	if attr, ok := content.Attributes["description"]; ok {
		hclDiags := gohcl.DecodeExpression(attr.Expr, nil, &ret.Description)
		diags = diags.Append(hclDiags)
	}
	if attr, ok := content.Attributes["sensitive"]; ok {
		hclDiags := gohcl.DecodeExpression(attr.Expr, nil, &ret.Sensitive)
		diags = diags.Append(hclDiags)
	}
	if attr, ok := content.Attributes["ephemeral"]; ok {
		hclDiags := gohcl.DecodeExpression(attr.Expr, nil, &ret.Ephemeral)
		diags = diags.Append(hclDiags)
	}

	for _, block := range content.Blocks {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Custom variable validation not yet supported",
			Detail:   "Input variables for a stack configuration do not yet support custom variable validation.",
			Subject:  block.DefRange.Ptr(),
		})
	}

	return ret, diags
}

var inputVariableBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "type", Required: true},
		{Name: "default", Required: false},
		{Name: "description", Required: false},
		{Name: "sensitive", Required: false},
		{Name: "ephemeral", Required: false},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "validation"},
	},
}

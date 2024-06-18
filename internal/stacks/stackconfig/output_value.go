// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// OutputValue is a declaration of a result from a stack configuration, which
// can be read by the stack's caller.
type OutputValue struct {
	Name string

	Type TypeConstraint

	Value       hcl.Expression
	Description string
	Sensitive   bool
	Ephemeral   bool

	DeclRange tfdiags.SourceRange
}

func decodeOutputValueBlock(block *hcl.Block) (*OutputValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &OutputValue{
		Name:      block.Labels[0],
		DeclRange: tfdiags.SourceRangeFromHCL(block.DefRange),
	}
	if !hclsyntax.ValidIdentifier(ret.Name) {
		diags = diags.Append(invalidNameDiagnostic(
			"Invalid name for output value",
			block.LabelRanges[0],
		))
		return nil, diags
	}

	content, hclDiags := block.Body.Content(outputValueBlockSchema)
	diags = diags.Append(hclDiags)

	if attr, ok := content.Attributes["type"]; ok {
		ret.Type.Expression = attr.Expr
	}
	if attr, ok := content.Attributes["value"]; ok {
		ret.Value = attr.Expr
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
			Summary:  "Preconditions not yet supported",
			Detail:   "Output values for a stack configuration do not yet support preconditions.",
			Subject:  block.DefRange.Ptr(),
		})
	}

	return ret, diags
}

var outputValueBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "type", Required: true},
		{Name: "value", Required: false},
		{Name: "description", Required: false},
		{Name: "sensitive", Required: false},
		{Name: "ephemeral", Required: false},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "precondition"},
	},
}

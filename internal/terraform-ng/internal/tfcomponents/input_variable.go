package tfcomponents

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/internal/typeexpr"
)

type InputVariable struct {
	Name           string
	TypeConstraint cty.Type
	Default        cty.Value

	DeclRange tfdiags.SourceRange
}

func (v *InputVariable) LocalAddr() addrs.InputVariable {
	return addrs.InputVariable{Name: v.Name}
}

func decodeInputVariableBlock(block *hcl.Block) (*InputVariable, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &InputVariable{
		Name:      block.Labels[0],
		DeclRange: tfdiags.SourceRangeFromHCL(block.DefRange),
	}
	if !hclsyntax.ValidIdentifier(block.Labels[0]) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid component group name",
			Detail:   fmt.Sprintf("Cannot use %q as a component group name: must be a valid identifier.", block.Labels[0]),
			Subject:  block.LabelRanges[0].Ptr(),
		})
	}

	content, hclDiags := block.Body.Content(variableBlockSchema)
	diags = diags.Append(hclDiags)

	if attr, ok := content.Attributes["type"]; ok {
		ty, hclDiags := typeexpr.TypeConstraint(attr.Expr)
		diags = diags.Append(hclDiags)
		if ty == cty.NilType {
			ty = cty.DynamicPseudoType
		}
		ret.TypeConstraint = ty
	}

	if attr, ok := content.Attributes["default"]; ok {
		rawV, hclDiags := attr.Expr.Value(nil)
		diags = diags.Append(hclDiags)
		if rawV == cty.NilVal {
			rawV = cty.DynamicVal
		}

		v, err := convert.Convert(rawV, ret.TypeConstraint)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid default value for variable",
				Detail:   fmt.Sprintf("The default value for variable %q does not conform to its type constraint: %s.", ret.Name, err),
				Subject:  attr.Expr.Range().Ptr(),
			})
			v = cty.DynamicVal
		}

		ret.Default = v
	}

	for _, block := range content.Blocks {
		switch block.Type {
		case "validation":
			// TODO: decode validation blocks
		default:
			panic(fmt.Sprintf("unexpected block type %q", block.Type))
		}
	}

	return ret, diags
}

var variableBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "type", Required: true},
		{Name: "default"},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "validation"},
	},
}

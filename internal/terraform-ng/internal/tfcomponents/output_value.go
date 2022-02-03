package tfcomponents

import (
	"fmt"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type OutputValue struct {
	Name  string
	Value hcl.Expression

	DeclRange tfdiags.SourceRange
}

func (v *OutputValue) LocalAddr() addrs.OutputValue {
	return addrs.OutputValue{Name: v.Name}
}

func decodeOutputValueBlock(block *hcl.Block) (*OutputValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &OutputValue{
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

	content, hclDiags := block.Body.Content(outputBlockSchema)
	diags = diags.Append(hclDiags)

	if attr, ok := content.Attributes["value"]; ok {
		ret.Value = attr.Expr
	}

	if _, ok := content.Attributes["sensitive"]; ok {
		// TODO
	}

	return ret, diags
}

var outputBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "value", Required: true},
		{Name: "sensitive"},
	},
}

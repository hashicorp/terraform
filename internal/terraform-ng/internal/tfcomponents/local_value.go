package tfcomponents

import (
	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type LocalValue struct {
	Name  string
	Value hcl.Expression

	DeclRange tfdiags.SourceRange
}

func (v *LocalValue) LocalAddr() addrs.LocalValue {
	return addrs.LocalValue{Name: v.Name}
}

func decodeLocalValuesBlock(block *hcl.Block) ([]*LocalValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var ret []*LocalValue
	attrs, hclDiags := block.Body.JustAttributes()
	diags = diags.Append(hclDiags)
	if len(attrs) == 0 {
		return ret, diags
	}

	ret = make([]*LocalValue, 0, len(attrs))
	for name, attr := range attrs {
		ret = append(ret, &LocalValue{
			Name:      name,
			Value:     attr.Expr,
			DeclRange: tfdiags.SourceRangeFromHCL(attr.NameRange),
		})
	}
	return ret, diags
}

package configs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
)

type MovedAction struct {
	From *addrs.Target
	To   *addrs.Target

	DeclRange hcl.Range
}

func decodeMovedBlock(block *hcl.Block) (*MovedAction, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	ma := &MovedAction{
		DeclRange: block.DefRange,
	}

	content, moreDiags := block.Body.Content(movedBlockSchema)
	diags = append(diags, moreDiags...)

	if attr, exists := content.Attributes["from"]; exists {
		traversal, traversalDiags := hcl.AbsTraversalForExpr(attr.Expr)
		diags = append(diags, traversalDiags...)
		if !traversalDiags.HasErrors() {
			from, fromDiags := addrs.ParseTarget(traversal)
			diags = append(diags, fromDiags.ToHCL()...)
			ma.From = from
		}
	}

	if attr, exists := content.Attributes["to"]; exists {
		traversal, traversalDiags := hcl.AbsTraversalForExpr(attr.Expr)
		diags = append(diags, traversalDiags...)
		if !traversalDiags.HasErrors() {
			to, toDiags := addrs.ParseTarget(traversal)
			diags = append(diags, toDiags.ToHCL()...)
			ma.To = to
		}
	}

	return ma, diags
}

var movedBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name:     "from",
			Required: true,
		},
		{
			Name:     "to",
			Required: true,
		},
	},
}

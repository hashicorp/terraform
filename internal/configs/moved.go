package configs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
)

type Moved struct {
	From *addrs.MoveEndpoint
	To   *addrs.MoveEndpoint

	DeclRange hcl.Range
}

func decodeMovedBlock(block *hcl.Block) (*Moved, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	moved := &Moved{
		DeclRange: block.DefRange,
	}

	content, moreDiags := block.Body.Content(movedBlockSchema)
	diags = append(diags, moreDiags...)

	if attr, exists := content.Attributes["from"]; exists {
		from, traversalDiags := hcl.AbsTraversalForExpr(attr.Expr)
		diags = append(diags, traversalDiags...)
		if !traversalDiags.HasErrors() {
			from, fromDiags := addrs.ParseMoveEndpoint(from)
			diags = append(diags, fromDiags.ToHCL()...)
			moved.From = from
		}
	}

	if attr, exists := content.Attributes["to"]; exists {
		to, traversalDiags := hcl.AbsTraversalForExpr(attr.Expr)
		diags = append(diags, traversalDiags...)
		if !traversalDiags.HasErrors() {
			to, toDiags := addrs.ParseMoveEndpoint(to)
			diags = append(diags, toDiags.ToHCL()...)
			moved.To = to
		}
	}

	// we can only move from a module to a module, resource to resource, etc.
	if !diags.HasErrors() {
		if !moved.From.MightUnifyWith(moved.To) {
			// We can catch some obviously-wrong combinations early here,
			// but we still have other dynamic validation to do at runtime.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid \"moved\" addresses",
				Detail:   "The \"from\" and \"to\" addresses must either both refer to resources or both refer to modules.",
				Subject:  &moved.DeclRange,
			})
		}
	}

	return moved, diags
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

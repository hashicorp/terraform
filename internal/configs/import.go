package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

type Import struct {
	ID string
	To addrs.AbsResourceInstance

	DeclRange hcl.Range
}

func decodeImportBlock(block *hcl.Block) (*Import, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	imp := &Import{
		DeclRange: block.DefRange,
	}

	content, moreDiags := block.Body.Content(importBlockSchema)
	diags = append(diags, moreDiags...)

	if attr, exists := content.Attributes["id"]; exists {
		id, idDiags := decodeId(attr.Expr)
		diags = append(diags, idDiags...)
		imp.ID = id
	}

	if attr, exists := content.Attributes["to"]; exists {
		to, traversalDiags := hcl.AbsTraversalForExpr(attr.Expr)
		diags = append(diags, traversalDiags...)
		if !traversalDiags.HasErrors() {
			to, toDiags := addrs.ParseAbsResourceInstance(to)
			diags = append(diags, toDiags.ToHCL()...)
			imp.To = to
			if !toDiags.HasErrors() && to.Resource.Resource.Mode != addrs.ManagedResourceMode {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid Resource",
					Detail:   fmt.Sprintf("%v is not a managed resource. Importing into a data source is not allowed.", to),
					Subject:  attr.Expr.Range().Ptr(),
				})
			}
		}
	}

	return imp, diags

}

func decodeId(expr hcl.Expression) (string, hcl.Diagnostics) {
	id, diags := expr.Value(nil)
	if diags.HasErrors() {
		return "", diags
	}
	if id.Type() != cty.String {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid Attribute",
			Detail:   fmt.Sprintf("Invalid attribute value for import id: %#v", id),
			Subject:  expr.Range().Ptr(),
		})
		return "", diags
	}
	return id.AsString(), diags
}

var importBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name:     "id",
			Required: true,
		},
		{
			Name:     "to",
			Required: true,
		},
	},
}

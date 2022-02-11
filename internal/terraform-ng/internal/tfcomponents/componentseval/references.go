package componentseval

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/terraform-ng/internal/ngaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func ExpressionReferences(expr hcl.Expression) ([]*ngaddrs.Reference, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var ret []*ngaddrs.Reference
	for _, traversal := range expr.Variables() {
		ref, moreDiags := ngaddrs.ParseRef(traversal)
		diags = diags.Append(moreDiags)
		if ref != nil {
			ret = append(ret, ref)
		}
	}
	return ret, diags
}

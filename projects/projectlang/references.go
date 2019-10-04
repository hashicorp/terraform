package projectlang

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/addrs"
)

func findReferencesInExpr(expr hcl.Expression) []*addrs.ProjectConfigReference {
	traversals := expr.Variables()
	if len(traversals) == 0 {
		return nil
	}
	ret := make([]*addrs.ProjectConfigReference, 0, len(traversals))
	for _, traversal := range expr.Variables() {
		ref, diags := addrs.ParseProjectConfigRef(traversal)
		if diags.HasErrors() {
			continue
		}
		ret = append(ret, ref)
	}
	return ret
}

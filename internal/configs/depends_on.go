// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"github.com/hashicorp/hcl/v2"
)

func DecodeDependsOn(attr *hcl.Attribute) ([]hcl.Traversal, hcl.Diagnostics) {
	var ret []hcl.Traversal
	exprs, diags := hcl.ExprList(attr.Expr)

	for _, expr := range exprs {
		expr, shimDiags := shimTraversalInString(expr, false)
		diags = append(diags, shimDiags...)

		traversal, travDiags := hcl.AbsTraversalForExpr(expr)
		diags = append(diags, travDiags...)

		if len(traversal) != 0 {
			if traversal.RootName() == "action" {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid depends_on Action Reference",
					Detail:   "The depends_on attribute cannot reference action blocks directly. You must reference a resource or data source instead.",
					Subject:  expr.Range().Ptr(),
				})
			} else {
				ret = append(ret, traversal)
			}
		}
	}

	return ret, diags
}

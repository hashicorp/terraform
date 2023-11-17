// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// LocalValue is a declaration of a private local value within a particular
// stack configuration. These are visible only within the scope of a particular
// [Stack].
type LocalValue struct {
	Name  string
	Value hcl.Expression

	DeclRange tfdiags.SourceRange
}

func decodeLocalValuesBlock(block *hcl.Block) ([]*LocalValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	attrs, hclDiags := block.Body.JustAttributes()
	diags = diags.Append(hclDiags)
	if len(attrs) == 0 {
		return nil, diags
	}

	ret := make([]*LocalValue, 0, len(attrs))
	for name, attr := range attrs {
		v := &LocalValue{
			Name:      name,
			Value:     attr.Expr,
			DeclRange: tfdiags.SourceRangeFromHCL(attr.NameRange),
		}
		if !hclsyntax.ValidIdentifier(v.Name) {
			diags = diags.Append(invalidNameDiagnostic(
				"Invalid name for local value",
				attr.NameRange,
			))
			continue
		}
		ret = append(ret, v)
	}
	return ret, diags
}

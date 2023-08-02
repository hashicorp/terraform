// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
)

type Import struct {
	ID hcl.Expression
	To addrs.AbsResourceInstance

	ProviderConfigRef *ProviderConfigRef
	Provider          addrs.Provider

	DeclRange         hcl.Range
	ProviderDeclRange hcl.Range
}

func decodeImportBlock(block *hcl.Block) (*Import, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	imp := &Import{
		DeclRange: block.DefRange,
	}

	content, moreDiags := block.Body.Content(importBlockSchema)
	diags = append(diags, moreDiags...)

	if attr, exists := content.Attributes["id"]; exists {
		imp.ID = attr.Expr
	}

	if attr, exists := content.Attributes["to"]; exists {
		traversal, traversalDiags := hcl.AbsTraversalForExpr(attr.Expr)
		diags = append(diags, traversalDiags...)
		if !traversalDiags.HasErrors() {
			to, toDiags := addrs.ParseAbsResourceInstance(traversal)
			diags = append(diags, toDiags.ToHCL()...)
			imp.To = to
		}
	}

	if attr, exists := content.Attributes["provider"]; exists {
		if len(imp.To.Module) > 0 {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid import provider argument",
				Detail:   "The provider argument can only be specified in import blocks that will generate configuration.\n\nUse the providers argument within the module block to configure providers for all resources within a module, including imported resources.",
				Subject:  attr.Range.Ptr(),
			})
		}

		var providerDiags hcl.Diagnostics
		imp.ProviderConfigRef, providerDiags = decodeProviderConfigRef(attr.Expr, "provider")
		imp.ProviderDeclRange = attr.Range
		diags = append(diags, providerDiags...)
	}

	return imp, diags
}

var importBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "provider",
		},
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

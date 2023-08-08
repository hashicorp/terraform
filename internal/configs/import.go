// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Import struct {
	ID hcl.Expression

	To hcl.Expression
	// The To address may not be resolvable immediately if it contains dynamic
	// index expressions, so we will extract the ConfigResource address and
	// store it here for reference.
	ToResource addrs.ConfigResource

	ForEach hcl.Expression

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
		imp.To = attr.Expr

		addr, toDiags := parseConfigResourceFromExpression(attr.Expr)
		diags = diags.Extend(toDiags.ToHCL())
		imp.ToResource = addr
	}

	if attr, exists := content.Attributes["for_each"]; exists {
		imp.ForEach = attr.Expr
	}

	if attr, exists := content.Attributes["provider"]; exists {
		if len(imp.ToResource.Module) > 0 {
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
			Name: "for_each",
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

// parseResourceInstanceFromExpression takes an arbitrary expression
// representing a resource instance, and parses out the static ConfigResource
// skipping an variable index expressions. This is used to connect an import
// block's "to" to the configuration address before the full instance
// expressions are evaluated.
func parseConfigResourceFromExpression(expr hcl.Expression) (addrs.ConfigResource, tfdiags.Diagnostics) {
	traversal, hcdiags := exprToResourceTraversal(expr)
	if hcdiags.HasErrors() {
		return addrs.ConfigResource{}, tfdiags.Diagnostics(nil).Append(hcdiags)
	}

	addr, diags := addrs.ParseAbsResourceInstance(traversal)
	if diags.HasErrors() {
		return addrs.ConfigResource{}, diags
	}

	return addr.ConfigResource(), diags
}

// exprToResourceTraversal is used to parse the import block's to expression,
// which must be a resource instance, but may contain limited variables with
// index expressions. Since we only need the ConfigResource to connect the
// import to the configuration, we skip any index expressions.
func exprToResourceTraversal(expr hcl.Expression) (hcl.Traversal, hcl.Diagnostics) {
	var trav hcl.Traversal
	var diags hcl.Diagnostics

	switch e := expr.(type) {
	case *hclsyntax.RelativeTraversalExpr:
		t, d := exprToResourceTraversal(e.Source)
		diags = diags.Extend(d)
		trav = append(trav, t...)
		trav = append(trav, e.Traversal...)

	case *hclsyntax.ScopeTraversalExpr:
		// a static reference, we can just append the traversal
		trav = append(trav, e.Traversal...)

	case *hclsyntax.IndexExpr:
		// Get the collection from the index expression, we don't need the
		// index for a ConfigResource
		t, d := exprToResourceTraversal(e.Collection)
		diags = diags.Extend(d)
		if diags.HasErrors() {
			return nil, diags
		}
		trav = append(trav, t...)

	default:
		// if we don't recognise the expression type (which means we are likely
		// dealing with a test mock), try and interpret this as an absolute
		// traversal
		t, d := hcl.AbsTraversalForExpr(e)
		diags = diags.Extend(d)
		trav = append(trav, t...)
	}

	return trav, diags
}

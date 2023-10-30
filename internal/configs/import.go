// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	hcljson "github.com/hashicorp/hcl/v2/json"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
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
		toExpr, jsDiags := unwrapJSONRefExpr(attr.Expr)
		diags = diags.Extend(jsDiags)
		if diags.HasErrors() {
			return imp, diags
		}

		imp.To = toExpr

		addr, toDiags := parseConfigResourceFromExpression(imp.To)
		diags = diags.Extend(toDiags.ToHCL())

		if addr.Resource.Mode != addrs.ManagedResourceMode {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid import address",
				Detail:   "Only managed resources can be imported.",
				Subject:  attr.Range.Ptr(),
			})
		}

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

// unwrapJSONRefExpr takes a string expression from a JSON configuration,
// and re-evaluates the string as HCL. If the expression is not JSON, the
// original expression is returned directly.
func unwrapJSONRefExpr(expr hcl.Expression) (hcl.Expression, hcl.Diagnostics) {
	if !hcljson.IsJSONExpression(expr) {
		return expr, nil
	}

	// We can abuse the hcl json api and rely on the fact that calling
	// Value on a json expression with no EvalContext will return the
	// raw string. We can then parse that as normal hcl syntax, and
	// continue with the decoding.
	v, diags := expr.Value(nil)
	if diags.HasErrors() {
		return nil, diags
	}

	// the JSON representation can only be a string
	if v.Type() != cty.String {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference expression",
			Detail:   "A single reference string is required.",
			Subject:  expr.Range().Ptr(),
		})

		return nil, diags
	}

	rng := expr.Range()
	expr, ds := hclsyntax.ParseExpression([]byte(v.AsString()), rng.Filename, rng.Start)
	diags = diags.Extend(ds)
	return expr, diags
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

// parseImportToStatic attempts to parse the To address of an import block
// statically to get the resource address. This returns false when the address
// cannot be parsed, which is usually a result of dynamic index expressions
// using for_each.
func parseImportToStatic(expr hcl.Expression) (addrs.AbsResourceInstance, bool) {
	// we may have a nil expression in some error cases, which we can just
	// false to avoid the parsing
	if expr == nil {
		return addrs.AbsResourceInstance{}, false
	}

	var toDiags tfdiags.Diagnostics
	traversal, hd := hcl.AbsTraversalForExpr(expr)
	toDiags = toDiags.Append(hd)
	to, td := addrs.ParseAbsResourceInstance(traversal)
	toDiags = toDiags.Append(td)
	return to, !toDiags.HasErrors()
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ParseRemovedFrom parses the "from" attribute of a "removed" block in a
// configuration and returns the address of the configuration object being
// removed.
//
// In addition to the address, this function also returns a traversal that
// represents the unparsed index within the from expression. Users can
// optionally specify a specific index of a component to target.
func ParseRemovedFrom(expr hcl.Expression) (Component, hcl.Expression, tfdiags.Diagnostics) {
	var component Component
	var diags tfdiags.Diagnostics

	traversal, index, hclDiags := exprToComponentTraversal(expr)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return component, index, diags
	}

	if len(traversal) < 2 {
		return component, index, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid 'from' attribute",
			Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
			Subject:  expr.Range().Ptr(),
		})
	}

	root, ok := traversal[0].(hcl.TraverseRoot)
	if !ok || root.Name != "component" {
		return component, index, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid 'from' attribute",
			Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
			Subject:  expr.Range().Ptr(),
		})
	}

	name, ok := traversal[1].(hcl.TraverseAttr)
	if !ok {
		return component, index, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid 'from' attribute",
			Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
			Subject:  expr.Range().Ptr(),
		})
	}
	component.Name = name.Name

	return component, index, diags
}

// exprToComponentTraversal converts an HCL expression into a traversal that
// represents the component being targeted. We have to handle parsing this
// ourselves because removed block from arguments can contain index expressions
// which are not supported by hcl.AbsTraversalForExpr.
func exprToComponentTraversal(expr hcl.Expression) (hcl.Traversal, hcl.Expression, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	switch e := expr.(type) {
	case *hclsyntax.IndexExpr:
		t, d := hcl.AbsTraversalForExpr(e.Collection)
		diags = diags.Extend(d)
		if d.HasErrors() {
			return nil, nil, diags
		}
		return t, e.Key, diags
	case *hclsyntax.RelativeTraversalExpr:

		// This is an expression of the form `component.component_name[each.key].attribute`.
		// This is invalid at the moment, as we only support direct component
		// references. We'll return our own diagnostic here.

		return nil, nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid 'from' attribute",
			Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
			Subject:  expr.Range().Ptr(),
		})

	default:

		// For anything else, just rely on the default traversal logic.

		t, d := hcl.AbsTraversalForExpr(expr)
		diags = diags.Extend(d)
		if d.HasErrors() {
			return nil, nil, diags
		}

		if len(t) < 2 {
			return nil, nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid 'from' attribute",
				Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
				Subject:  expr.Range().Ptr(),
			})
		}

		// For now, removed blocks only support direct component references.
		// ie. you can't target a resource within a component, the next check
		// ensures this is true.

		if len(t) > 3 {
			return nil, nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid 'from' attribute",
				Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
				Subject:  expr.Range().Ptr(),
			})
		}

		if len(t) == 2 {
			return t, nil, diags
		}

		if index, ok := t[2].(hcl.TraverseIndex); ok {
			return t[:2], hcl.StaticExpr(index.Key, index.SrcRange), diags
		}

		return nil, nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid 'from' attribute",
			Detail:   "The 'from' attribute must designate a component that has been removed, in the form `component.component_name` or `component.component_name[\"key\"].",
		})
	}
}

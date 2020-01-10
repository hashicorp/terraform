package configs

import (
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
)

// RequiredProvider represents a declaration of a dependency on a particular
// provider version without actually configuring that provider. This is used in
// child modules that expect a provider to be passed in from their parent.
//
// TODO: "Source" is a placeholder for an attribute that is not yet supported.
type RequiredProvider struct {
	Name        string
	Source      string // TODO
	Requirement VersionConstraint
}

// ProviderRequirements represents merged provider version constraints.
// VersionConstraints come from terraform.require_providers blocks and provider
// blocks.
type ProviderRequirements struct {
	Type               addrs.Provider
	VersionConstraints []VersionConstraint
}

func decodeRequiredProvidersBlock(block *hcl.Block) ([]*RequiredProvider, hcl.Diagnostics) {
	attrs, diags := block.Body.JustAttributes()
	var reqs []*RequiredProvider
	for name, attr := range attrs {
		expr, err := attr.Expr.Value(nil)
		if err != nil {
			diags = append(diags, err...)
		}

		switch {
		case expr.Type().IsPrimitiveType():
			vc, reqDiags := decodeVersionConstraint(attr)
			diags = append(diags, reqDiags...)
			reqs = append(reqs, &RequiredProvider{
				Name:        name,
				Requirement: vc,
			})
		case expr.Type().IsObjectType():
			if expr.Type().HasAttribute("version") {
				vc := VersionConstraint{
					DeclRange: attr.Range,
				}
				constraintStr := expr.GetAttr("version").AsString()
				constraints, err := version.NewConstraint(constraintStr)
				if err != nil {
					// NewConstraint doesn't return user-friendly errors, so we'll just
					// ignore the provided error and produce our own generic one.
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid version constraint",
						Detail:   "This string does not use correct version constraint syntax.",
						Subject:  attr.Expr.Range().Ptr(),
					})
					reqs = append(reqs, &RequiredProvider{Name: name})
					return reqs, diags
				}
				vc.Required = constraints
				reqs = append(reqs, &RequiredProvider{Name: name, Requirement: vc})
			}
			// No version
			reqs = append(reqs, &RequiredProvider{Name: name})
		default:
			// should not happen
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider_requirements syntax",
				Detail:   "provider_requirements entries must be strings or objects.",
				Subject:  attr.Expr.Range().Ptr(),
			})
			reqs = append(reqs, &RequiredProvider{Name: name})
			return reqs, diags
		}
	}
	return reqs, diags
}

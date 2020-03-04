package tfconfig

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty/gocty"
)

// ProviderRef is a reference to a provider configuration within a module.
// It represents the contents of a "provider" argument in a resource, or
// a value in the "providers" map for a module call.
type ProviderRef struct {
	Name  string `json:"name"`
	Alias string `json:"alias,omitempty"` // Empty if the default provider configuration is referenced
}

type ProviderRequirement struct {
	Source             string   `json:"source,omitempty"`
	VersionConstraints []string `json:"version_constraints,omitempty"`
}

func decodeRequiredProvidersBlock(block *hcl.Block) (map[string]*ProviderRequirement, hcl.Diagnostics) {
	attrs, diags := block.Body.JustAttributes()
	reqs := make(map[string]*ProviderRequirement)
	for name, attr := range attrs {
		expr, err := attr.Expr.Value(nil)
		if err != nil {
			diags = append(diags, err...)
		}

		switch {
		case expr.Type().IsPrimitiveType():
			var version string
			valDiags := gohcl.DecodeExpression(attr.Expr, nil, &version)
			diags = append(diags, valDiags...)
			if !valDiags.HasErrors() {
				reqs[name] = &ProviderRequirement{
					VersionConstraints: []string{version},
				}
			}

		case expr.Type().IsObjectType():
			var pr ProviderRequirement
			if expr.Type().HasAttribute("version") {
				var version string
				err := gocty.FromCtyValue(expr.GetAttr("version"), &version)
				if err == nil {
					pr.VersionConstraints = append(pr.VersionConstraints, version)
				} else {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Unsuitable value type",
						Detail:   "Unsuitable value: string required",
						Subject:  attr.Expr.Range().Ptr(),
					})
				}
			}
			if expr.Type().HasAttribute("source") {
				var source string
				err := gocty.FromCtyValue(expr.GetAttr("source"), &source)
				if err == nil {
					pr.Source = source
				} else {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Unsuitable value type",
						Detail:   "Unsuitable value: string required",
						Subject:  attr.Expr.Range().Ptr(),
					})
				}
			}
			reqs[name] = &pr

		default:
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unsuitable value type",
				Detail:   "Unsuitable value: string required",
				Subject:  attr.Expr.Range().Ptr(),
			})
		}
	}

	return reqs, diags
}

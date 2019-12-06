package configs

import (
	"github.com/hashicorp/hcl/v2"
)

// ProviderRequirement represents a declaration of a dependency on a particular
// provider version without actually configuring that provider. This is used in
// child modules that expect a provider to be passed in from their parent.
//
// TODO: "Source" is a placeholder for an attribute that is not yet supported.
type ProviderRequirement struct {
	Name        string
	Source      string // TODO
	Requirement VersionConstraint
}

func decodeRequiredProvidersBlock(block *hcl.Block) ([]*ProviderRequirement, hcl.Diagnostics) {
	attrs, diags := block.Body.JustAttributes()
	var reqs []*ProviderRequirement
	for name, attr := range attrs {
		req, reqDiags := decodeVersionConstraint(attr)
		diags = append(diags, reqDiags...)
		if !diags.HasErrors() {
			reqs = append(reqs, &ProviderRequirement{
				Name:        name,
				Requirement: req,
			})
		}
	}
	return reqs, diags
}

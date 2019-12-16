package configs

import (
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
		req, reqDiags := decodeVersionConstraint(attr)
		diags = append(diags, reqDiags...)
		if !diags.HasErrors() {
			reqs = append(reqs, &RequiredProvider{
				Name:        name,
				Requirement: req,
			})
		}
	}
	return reqs, diags
}

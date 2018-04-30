package configs

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
)

// Provider represents a "provider" block in a module or file. A provider
// block is a provider configuration, and there can be zero or more
// configurations for each actual provider.
type Provider struct {
	Name       string
	NameRange  hcl.Range
	Alias      string
	AliasRange *hcl.Range // nil if no alias set

	Version VersionConstraint

	Config hcl.Body

	DeclRange hcl.Range
}

func decodeProviderBlock(block *hcl.Block) (*Provider, hcl.Diagnostics) {
	content, config, diags := block.Body.PartialContent(providerBlockSchema)

	provider := &Provider{
		Name:      block.Labels[0],
		NameRange: block.LabelRanges[0],
		Config:    config,
		DeclRange: block.DefRange,
	}

	if attr, exists := content.Attributes["alias"]; exists {
		valDiags := gohcl.DecodeExpression(attr.Expr, nil, &provider.Alias)
		diags = append(diags, valDiags...)
		provider.AliasRange = attr.Expr.Range().Ptr()

		if !hclsyntax.ValidIdentifier(provider.Alias) {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid provider configuration alias",
				Detail:   fmt.Sprintf("An alias must be a valid name. %s", badIdentifierDetail),
			})
		}
	}

	if attr, exists := content.Attributes["version"]; exists {
		var versionDiags hcl.Diagnostics
		provider.Version, versionDiags = decodeVersionConstraint(attr)
		diags = append(diags, versionDiags...)
	}

	return provider, diags
}

// Addr returns the address of the receiving provider configuration, relative
// to its containing module.
func (p *Provider) Addr() addrs.ProviderConfig {
	return addrs.ProviderConfig{
		Type:  p.Name,
		Alias: p.Alias,
	}
}

func (p *Provider) moduleUniqueKey() string {
	if p.Alias != "" {
		return fmt.Sprintf("%s.%s", p.Name, p.Alias)
	}
	return p.Name
}

// ProviderRequirement represents a declaration of a dependency on a particular
// provider version without actually configuring that provider. This is used in
// child modules that expect a provider to be passed in from their parent.
type ProviderRequirement struct {
	Name        string
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

var providerBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "alias",
		},
		{
			Name: "version",
		},
	},
}

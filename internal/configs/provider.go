// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
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

	// TODO: this may not be set in some cases, so it is not yet suitable for
	// use outside of this package. We currently only use it for internal
	// validation, but once we verify that this can be set in all cases, we can
	// export this so providers don't need to be re-resolved.
	// This same field is also added to the ProviderConfigRef struct.
	providerType addrs.Provider

	// Mock and MockData declare this provider as a "mock_provider", which means
	// it should use the data in MockData instead of actually initialising the
	// provider.
	Mock     bool
	MockData *MockData

	// MockDataExternalSource is a file path pointing to the external data
	// file for a mock provider. An empty string indicates all data should be
	// loaded inline.
	MockDataExternalSource string
}

func decodeProviderBlock(block *hcl.Block, testFile bool) (*Provider, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	content, config, moreDiags := block.Body.PartialContent(providerBlockSchema)
	diags = append(diags, moreDiags...)

	// Provider names must be localized. Produce an error with a message
	// indicating the action the user can take to fix this message if the local
	// name is not localized.
	name := block.Labels[0]
	nameDiags := checkProviderNameNormalized(name, block.DefRange)
	diags = append(diags, nameDiags...)
	if nameDiags.HasErrors() {
		// If the name is invalid then we mustn't produce a result because
		// downstreams could try to use it as a provider type and then crash.
		return nil, diags
	}

	provider := &Provider{
		Name:      name,
		NameRange: block.LabelRanges[0],
		Config:    config,
		DeclRange: block.DefRange,

		// We'll just explicitly mark real providers as not being mocks even
		// though this is the default.
		Mock: false,
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
		if testFile {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Version constraints are not allowed in test files",
				Detail:   "Version constraints inside provider configuration blocks are not allowed in test files. To silence this error, move the provider version constraint into the required_providers block of the configuration that uses this provider.",
				Subject:  attr.Expr.Range().Ptr(),
			})
		} else {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Version constraints inside provider configuration blocks are deprecated",
				Detail:   "Terraform 0.13 and earlier allowed provider version constraints inside the provider configuration block, but that is now deprecated and will be removed in a future version of Terraform. To silence this warning, move the provider version constraint into the required_providers block.",
				Subject:  attr.Expr.Range().Ptr(),
			})
			var versionDiags hcl.Diagnostics
			provider.Version, versionDiags = decodeVersionConstraint(attr)
			diags = append(diags, versionDiags...)
		}
	}

	// Reserved attribute names
	for _, name := range []string{"count", "depends_on", "for_each", "source"} {
		if attr, exists := content.Attributes[name]; exists {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reserved argument name in provider block",
				Detail:   fmt.Sprintf("The provider argument name %q is reserved for use by Terraform in a future version.", name),
				Subject:  &attr.NameRange,
			})
		}
	}

	var seenEscapeBlock *hcl.Block
	for _, block := range content.Blocks {
		switch block.Type {
		case "_":
			if seenEscapeBlock != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate escaping block",
					Detail: fmt.Sprintf(
						"The special block type \"_\" can be used to force particular arguments to be interpreted as provider-specific rather than as meta-arguments, but each provider block can have only one such block. The first escaping block was at %s.",
						seenEscapeBlock.DefRange,
					),
					Subject: &block.DefRange,
				})
				continue
			}
			seenEscapeBlock = block

			// When there's an escaping block its content merges with the
			// existing config we extracted earlier, so later decoding
			// will see a blend of both.
			provider.Config = hcl.MergeBodies([]hcl.Body{provider.Config, block.Body})

		default:
			// All of the other block types in our schema are reserved for
			// future expansion.
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reserved block type name in provider block",
				Detail:   fmt.Sprintf("The block type name %q is reserved for use by Terraform in a future version.", block.Type),
				Subject:  &block.TypeRange,
			})
		}
	}

	return provider, diags
}

// Addr returns the address of the receiving provider configuration, relative
// to its containing module.
func (p *Provider) Addr() addrs.LocalProviderConfig {
	return addrs.LocalProviderConfig{
		LocalName: p.Name,
		Alias:     p.Alias,
	}
}

func (p *Provider) moduleUniqueKey() string {
	if p.Alias != "" {
		return fmt.Sprintf("%s.%s", p.Name, p.Alias)
	}
	return p.Name
}

// ParseProviderConfigCompact parses the given absolute traversal as a relative
// provider address in compact form. The following are examples of traversals
// that can be successfully parsed as compact relative provider configuration
// addresses:
//
//   - aws
//   - aws.foo
//
// This function will panic if given a relative traversal.
//
// If the returned diagnostics contains errors then the result value is invalid
// and must not be used.
func ParseProviderConfigCompact(traversal hcl.Traversal) (addrs.LocalProviderConfig, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := addrs.LocalProviderConfig{
		LocalName: traversal.RootName(),
	}

	if len(traversal) < 2 {
		// Just a type name, then.
		return ret, diags
	}

	aliasStep := traversal[1]
	switch ts := aliasStep.(type) {
	case hcl.TraverseAttr:
		ret.Alias = ts.Name
		return ret, diags
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider configuration address",
			Detail:   "The provider type name must either stand alone or be followed by an alias name separated with a dot.",
			Subject:  aliasStep.SourceRange().Ptr(),
		})
	}

	if len(traversal) > 2 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider configuration address",
			Detail:   "Extraneous extra operators after provider configuration address.",
			Subject:  traversal[2:].SourceRange().Ptr(),
		})
	}

	return ret, diags
}

// ParseProviderConfigCompactStr is a helper wrapper around ParseProviderConfigCompact
// that takes a string and parses it with the HCL native syntax traversal parser
// before interpreting it.
//
// This should be used only in specialized situations since it will cause the
// created references to not have any meaningful source location information.
// If a reference string is coming from a source that should be identified in
// error messages then the caller should instead parse it directly using a
// suitable function from the HCL API and pass the traversal itself to
// ParseProviderConfigCompact.
//
// Error diagnostics are returned if either the parsing fails or the analysis
// of the traversal fails. There is no way for the caller to distinguish the
// two kinds of diagnostics programmatically. If error diagnostics are returned
// then the returned address is invalid.
func ParseProviderConfigCompactStr(str string) (addrs.LocalProviderConfig, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(str), "", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return addrs.LocalProviderConfig{}, diags
	}

	addr, addrDiags := ParseProviderConfigCompact(traversal)
	diags = diags.Append(addrDiags)
	return addr, diags
}

var providerBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "alias",
		},
		{
			Name: "version",
		},

		// Attribute names reserved for future expansion.
		{Name: "count"},
		{Name: "depends_on"},
		{Name: "for_each"},
		{Name: "source"},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "_"}, // meta-argument escaping block

		// The rest of these are reserved for future expansion.
		{Type: "lifecycle"},
		{Type: "locals"},
	},
}

// checkProviderNameNormalized verifies that the given string is already
// normalized and returns an error if not.
func checkProviderNameNormalized(name string, declrange hcl.Range) hcl.Diagnostics {
	var diags hcl.Diagnostics
	// verify that the provider local name is normalized
	normalized, err := addrs.IsProviderPartNormalized(name)
	if err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider local name",
			Detail:   fmt.Sprintf("%s is an invalid provider local name: %s", name, err),
			Subject:  &declrange,
		})
		return diags
	}
	if !normalized {
		// we would have returned this error already
		normalizedProvider, _ := addrs.ParseProviderPart(name)
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider local name",
			Detail:   fmt.Sprintf("Provider names must be normalized. Replace %q with %q to fix this error.", name, normalizedProvider),
			Subject:  &declrange,
		})
	}
	return diags
}

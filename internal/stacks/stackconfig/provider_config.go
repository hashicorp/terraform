// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProviderConfig is a provider configuration declared within a [Stack].
type ProviderConfig struct {
	LocalAddr addrs.LocalProviderConfig

	// ProviderAddr is populated only when loaded through either
	// [LoadSingleStackConfig] or [LoadConfigDir], and contains the
	// fully-qualified provider address corresponding to the local name
	// given in field LocalAddr.
	ProviderAddr addrs.Provider

	// TODO: Figure out how we're going to retain the relevant subset of
	// a provider configuration in the state so that we still have what
	// we need to destroy any associated objects when a provider is removed
	// from the configuration.
	ForEach hcl.Expression

	// Config is the body of the nested block containing the provider-specific
	// configuration arguments, if specified. Some providers do not require
	// explicit arguments and so the nested block is optional; this field
	// will be nil if no block was included.
	Config hcl.Body

	ProviderNameRange tfdiags.SourceRange
	DeclRange         tfdiags.SourceRange
}

func decodeProviderConfigBlock(block *hcl.Block) (*ProviderConfig, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &ProviderConfig{
		LocalAddr: addrs.LocalProviderConfig{
			LocalName: block.Labels[0],

			// we call this "name" in the stacks configuration language,
			// but it's "Alias" here because we're reusing an address type
			// made for the Terraform module language.
			Alias: block.Labels[1],
		},
		ProviderNameRange: tfdiags.SourceRangeFromHCL(block.LabelRanges[0]),
		DeclRange:         tfdiags.SourceRangeFromHCL(block.DefRange),
	}
	if !hclsyntax.ValidIdentifier(ret.LocalAddr.LocalName) {
		diags = diags.Append(invalidNameDiagnostic(
			"Invalid provider local name",
			block.LabelRanges[0],
		))
		return nil, diags
	}
	if !hclsyntax.ValidIdentifier(ret.LocalAddr.Alias) {
		diags = diags.Append(invalidNameDiagnostic(
			"Invalid provider configuration name",
			block.LabelRanges[0],
		))
		return nil, diags
	}

	content, hclDiags := block.Body.Content(providerConfigBlockSchema)
	diags = diags.Append(hclDiags)

	if attr, ok := content.Attributes["for_each"]; ok {
		ret.ForEach = attr.Expr
	}

	for _, block := range content.Blocks {
		switch block.Type {
		case "config":
			if ret.Config != nil {
				if !hclsyntax.ValidIdentifier(ret.LocalAddr.LocalName) {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate config block",
						Detail:   "A provider configuration block must contain only one nested \"config\" block.",
						Subject:  block.DefRange.Ptr(),
					})
					return nil, diags
				}
				continue
			}
			ret.Config = block.Body
		default:
			// Should not get here because the above should cover all
			// block types declared in the schema.
			panic(fmt.Sprintf("unhandled block type %q", block.Type))
		}
	}

	return ret, diags
}

var providerConfigBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "for_each", Required: false},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "config"},
	},
}

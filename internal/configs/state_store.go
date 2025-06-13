// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"github.com/hashicorp/hcl/v2"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

// StateStore represents a "state_store" block inside a "terraform" block
// in a module or file.
type StateStore struct {
	Type   string
	Config hcl.Body

	Provider     *Provider
	ProviderAddr tfaddr.Provider

	TypeRange hcl.Range
	DeclRange hcl.Range
}

func decodeStateStoreBlock(block *hcl.Block) (*StateStore, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	ss := &StateStore{
		Type:      block.Labels[0],
		TypeRange: block.LabelRanges[0],
		Config:    block.Body,
		DeclRange: block.DefRange,
	}

	content, remain, moreDiags := block.Body.PartialContent(StateStorageBlockSchema)
	diags = append(diags, moreDiags...)
	ss.Config = remain

	if len(content.Blocks) == 0 {
		return nil, append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Missing provider block",
			Detail:   "A 'provider' block is required in 'state_store' blocks",
			Subject:  block.Body.MissingItemRange().Ptr(),
		})
	}
	if len(content.Blocks) > 1 {
		return nil, append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Duplicate provider block",
			Detail:   "Only one 'provider' block should be present in a 'state_store' block",
			Subject:  &content.Blocks[1].DefRange,
		})
	}

	providerBlock := content.Blocks[0]

	provider, providerDiags := decodeProviderBlock(providerBlock, false)
	if providerDiags.HasErrors() {
		return nil, append(diags, providerDiags...)
	}
	if provider.AliasRange != nil {
		// This block is in its own namespace in the state_store block; aliases are irrelevant
		return nil, append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unexpected provider alias",
			Detail:   "Aliases are disallowed in the 'provider' block in the 'state_store' block",
			Subject:  provider.AliasRange,
		})
	}

	ss.Provider = provider
	// We cannot set a value for ss.ProviderAddr at this point. This is done during BuildConfig logic, later.

	return ss, diags
}

var StateStorageBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "provider",
			LabelNames: []string{"type"},
		},
	},
}

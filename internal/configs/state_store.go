// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// StateStore represents a "state_store" block inside a "terraform" block
// in a module or file.
type StateStore struct {
	Type string

	// Config is the full configuration of the state_store block, including the
	// nested provider block. The nested provider block config is accessible
	// in isolation via (StateStore).Provider.Config
	Config hcl.Body

	Provider *Provider

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

// Hash produces a hash value for the receiver that covers the type and the
// portions of the config that conform to the state_store schema. The provider
// block that is nested inside state_store is ignored.
//
// If the config does not conform to the schema then the result is not
// meaningful for comparison since it will be based on an incomplete result.
//
// As an exception, required attributes in the schema are treated as optional
// for the purpose of hashing, so that an incomplete configuration can still
// be hashed. Other errors, such as extraneous attributes, have no such special
// case.
func (b *StateStore) Hash(stateStoreSchema *configschema.Block, providerSchema *configschema.Block) (stateStoreHash, providerHash int, diags tfdiags.Diagnostics) {

	// 1. Prepare the state_store hash

	// The state store schema should not include a provider block or attr
	if _, exists := stateStoreSchema.Attributes["provider"]; exists {
		return 0, 0, diags.Append(fmt.Errorf("error when creating hash of state_store config: schema contains a provider attribute. \nThis is a bug in the provider used for state storage, which should be reported in the provider's own issue tracker."))
	}
	if _, exists := stateStoreSchema.BlockTypes["provider"]; exists {
		return 0, 0, diags.Append(fmt.Errorf("error when creating hash of state_store config: schema contains a provider block. \nThis is a bug in the provider used for state storage, which should be reported in the provider's own issue tracker."))
	}

	// Don't fail if required attributes are not set. Instead, we'll just
	// hash them as nulls.
	schema := stateStoreSchema.NoneRequired()
	spec := schema.DecoderSpec()

	// The value `b.Config` will include data about the provider block nested inside state_store
	// so we need to ignore it. PartialDecode allows that 'extra' provider block to be ignored,
	// but we need to check that's the only thing being ignored.
	ssVal, decodeDiags := hcldec.Decode(b.Config, spec, nil)
	if decodeDiags.HasErrors() {
		for _, diag := range decodeDiags {
			if diag.Detail == "Blocks of type \"provider\" are not expected here." {
				// We want to tolerate this.
				continue
			}
			diags = diags.Append(diag)
		}
		if diags.HasErrors() {
			return 0, 0, diags
		}
	}

	// We're on the happy path, so continue to get the hash
	if ssVal == cty.NilVal {
		ssVal = cty.UnknownVal(schema.ImpliedType())
	}
	ssToHash := cty.TupleVal([]cty.Value{
		cty.StringVal(b.Type),
		ssVal,
	})

	// 2. Prepare the provider hash
	schema = providerSchema.NoneRequired()
	spec = schema.DecoderSpec()
	pVal, decodeDiags := hcldec.Decode(b.Provider.Config, spec, nil)
	if decodeDiags.HasErrors() {
		diags = diags.Append(decodeDiags)
		return 0, 0, diags
	}
	if pVal == cty.NilVal {
		pVal = cty.UnknownVal(schema.ImpliedType())
	}
	pToHash := cty.TupleVal([]cty.Value{
		cty.StringVal(b.Type),
		pVal,
	})

	return ssToHash.Hash(), pToHash.Hash(), diags
}

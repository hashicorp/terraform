// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty/cty"
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

// resolveStateStoreProviderType is used to obtain provider source data from required_providers data.
// This code is reused in code for parsing config and modules.
func resolveStateStoreProviderType(requiredProviders map[string]*RequiredProvider, stateStore StateStore) (tfaddr.Provider, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	// We intentionally don't look for entries in required_providers under different local names and match them
	// Users should use the same local name in the nested provider block as in required_providers.
	addr, foundReqProviderEntry := requiredProviders[stateStore.Provider.Name]
	switch {
	case !foundReqProviderEntry && stateStore.Provider.Name == "terraform":
		// We do not expect users to include built in providers in required_providers
		// So, if we don't find an entry in required_providers under local name 'terraform' we assume
		// that the builtin provider is intended.
		return tfaddr.Provider{
			Hostname:  tfaddr.BuiltInProviderHost,
			Namespace: tfaddr.BuiltInProviderNamespace,
			Type:      stateStore.Provider.Name,
		}, nil
	case !foundReqProviderEntry:
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Missing entry in required_providers",
			Detail: fmt.Sprintf("The provider used for state storage must have a matching entry in required_providers. Please add an entry for %s (%q)",
				stateStore.Provider.Name,
				stateStore.ProviderAddr),
			Subject: &stateStore.DeclRange,
		})
		return tfaddr.Provider{}, diags
	default:
		// We've got a required_providers entry to use
		return addr.Type, nil
	}
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
func (b *StateStore) Hash(schema *configschema.Block) int {
	// Don't fail if required attributes are not set. Instead, we'll just
	// hash them as nulls.
	schema = schema.NoneRequired()
	spec := schema.DecoderSpec()
	val, _ := hcldec.Decode(b.Config, spec, nil)
	if val == cty.NilVal {
		val = cty.UnknownVal(schema.ImpliedType())
	}

	toHash := cty.TupleVal([]cty.Value{
		cty.StringVal(b.Type),
		val,
	})

	return toHash.Hash()
}

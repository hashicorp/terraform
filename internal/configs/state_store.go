// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"
	"os"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/getproviders/reattach"
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
	// ProviderAddr contains the FQN of the provider used for pluggable state storage.
	// This is required for accessing provider factories during Terraform command logic,
	// and is used in diagnostics
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
	// We cannot set a value for ss.ProviderAddr at this point. Instead, this is done later when the
	// config has been parsed into a Config or Module and required_providers data is available.

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
// The only exception is the builtin terraform provider, which we return source data for without using required_providers.
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
		return addrs.NewBuiltInProvider("terraform"), nil
	case !foundReqProviderEntry:
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Missing entry in required_providers",
			Detail: fmt.Sprintf("The provider used for state storage must have a matching entry in required_providers. Please add an entry for provider %s",
				stateStore.Provider.Name,
			),
			Subject: &stateStore.DeclRange,
		})
		return tfaddr.Provider{}, diags
	default:
		// We've got a required_providers entry to use
		// This code path is used for both re-attached providers
		// providers that are fully managed by Terraform.
		return addr.Type, nil
	}
}

// Hash produces a hash value for the receiver that covers:
// 1) the portions of the config that conform to the state_store schema.
// 2) the portions of the config that conform to the provider schema.
// 3) the state store type
// 4) the provider source
// 5) the provider name
// 6) the provider version
//
// If the config does not conform to the schema then the result is not
// meaningful for comparison since it will be based on an incomplete result.
//
// As an exception, required attributes in the schema are treated as optional
// for the purpose of hashing, so that an incomplete configuration can still
// be hashed. Other errors, such as extraneous attributes, have no such special
// case.
func (b *StateStore) Hash(stateStoreSchema *configschema.Block, providerSchema *configschema.Block, stateStoreProviderVersion *version.Version) (int, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// 1. Prepare the state_store hash

	// The state store schema should not include a provider block or attr
	if _, exists := stateStoreSchema.Attributes["provider"]; exists {
		return 0, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Protected argument name \"provider\" in state store schema",
			Detail:   "Schemas for state stores cannot contain attributes or blocks called \"provider\", to avoid confusion with the provider block nested inside the state_store block. This is a bug in the provider used for state storage, which should be reported in the provider's own issue tracker.",
			Context:  &b.Provider.DeclRange,
		})
	}
	if _, exists := stateStoreSchema.BlockTypes["provider"]; exists {
		return 0, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Protected block name \"provider\" in state store schema",
			Detail:   "Schemas for state stores cannot contain attributes or blocks called \"provider\", to avoid confusion with the provider block nested inside the state_store block. This is a bug in the provider used for state storage, which should be reported in the provider's own issue tracker.",
			Context:  &b.Provider.DeclRange,
		})
	}

	// Don't fail if required attributes are not set. Instead, we'll just
	// hash them as nulls.
	schema := stateStoreSchema.NoneRequired()
	spec := schema.DecoderSpec()

	// The value `b.Config` will include data about the provider block nested inside state_store
	// so we need to ignore it. Decode will return errors about 'extra' attrs and blocks. We can ignore
	// the diagnostic reporting the unexpected provider block, but we need to handle all other diagnostics.
	// but we need to check that's the only thing being ignored.
	ssVal, decodeDiags := hcldec.Decode(b.Config, spec, nil)
	if decodeDiags.HasErrors() {
		for _, diag := range decodeDiags {
			diags = diags.Append(diag)
		}
		if diags.HasErrors() {
			return 0, diags
		}
	}

	// We're on the happy path, but handle if we got a nil value above
	if ssVal == cty.NilVal {
		ssVal = cty.UnknownVal(schema.ImpliedType())
	}

	// 2. Prepare the provider hash
	schema = providerSchema.NoneRequired()
	spec = schema.DecoderSpec()
	pVal, decodeDiags := hcldec.Decode(b.Provider.Config, spec, nil)
	if decodeDiags.HasErrors() {
		diags = diags.Append(decodeDiags)
		return 0, diags
	}
	if pVal == cty.NilVal {
		pVal = cty.UnknownVal(schema.ImpliedType())
	}

	var providerVersionString string
	if stateStoreProviderVersion == nil {
		isReattached, err := reattach.IsProviderReattached(b.ProviderAddr, os.Getenv("TF_REATTACH_PROVIDERS"))
		if err != nil {
			return 0, diags.Append(fmt.Errorf("Unable to determine if state storage provider is reattached while hashing state store configuration. This is a bug in Terraform and should be reported: %w", err))
		}

		if (b.ProviderAddr.Hostname != tfaddr.BuiltInProviderHost) &&
			!isReattached {
			return 0, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unable to calculate hash of state store configuration",
				Detail:   "Provider version data was missing during hash generation. This is a bug in Terraform and should be reported.",
			})
		}

		// Version information can be empty but only if the provider is builtin or unmanaged by Terraform
		providerVersionString = ""
	} else {
		providerVersionString = stateStoreProviderVersion.String()
	}

	toHash := cty.TupleVal([]cty.Value{
		cty.StringVal(b.Type), // state store type
		ssVal,                 // state store config

		cty.StringVal(b.ProviderAddr.String()), // provider source
		cty.StringVal(providerVersionString),   // provider version
		cty.StringVal(b.Provider.Name),         // provider name - this is directly parsed from the config, whereas provider source is added separately later after config is parsed.
		pVal,                                   // provider config
	})
	return toHash.Hash(), diags
}

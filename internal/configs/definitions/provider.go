// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
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
	ProviderType addrs.Provider

	// Mock and MockData declare this provider as a "mock_provider", which means
	// it should use the data in MockData instead of actually initialising the
	// provider. MockDataDuringPlan tells the provider that, by default, it
	// should generate values during the planning stage instead of waiting for
	// the apply stage.
	Mock               bool
	MockDataDuringPlan bool
	MockData           *MockData

	// MockDataExternalSource is a file path pointing to the external data
	// file for a mock provider. An empty string indicates all data should be
	// loaded inline.
	MockDataExternalSource string
}

// Addr returns the address of the receiving provider configuration, relative
// to its containing module.
func (p *Provider) Addr() addrs.LocalProviderConfig {
	return addrs.LocalProviderConfig{
		LocalName: p.Name,
		Alias:     p.Alias,
	}
}

// ModuleUniqueKey returns a unique key for this provider within a module.
func (p *Provider) ModuleUniqueKey() string {
	if p.Alias != "" {
		return fmt.Sprintf("%s.%s", p.Name, p.Alias)
	}
	return p.Name
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
)

// ProviderConfigRef represents a reference to a provider configuration.
type ProviderConfigRef struct {
	Name       string
	NameRange  hcl.Range
	Alias      string
	AliasRange *hcl.Range // nil if alias not set

	// TODO: this may not be set in some cases, so it is not yet suitable for
	// use outside of this package. We currently only use it for internal
	// validation, but once we verify that this can be set in all cases, we can
	// export this so providers don't need to be re-resolved.
	// This same field is also added to the Provider struct.
	ProviderType addrs.Provider
}

// Addr returns the provider config address corresponding to the receiving
// config reference.
//
// This is a trivial conversion, essentially just discarding the source
// location information and keeping just the addressing information.
func (r *ProviderConfigRef) Addr() addrs.LocalProviderConfig {
	return addrs.LocalProviderConfig{
		LocalName: r.Name,
		Alias:     r.Alias,
	}
}

func (r *ProviderConfigRef) String() string {
	if r == nil {
		return "<nil>"
	}
	if r.Alias != "" {
		return fmt.Sprintf("%s.%s", r.Name, r.Alias)
	}
	return r.Name
}

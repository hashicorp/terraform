package stackconfig

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
)

// ProviderConfig is a provider configuration declared within a [Stack].
type ProviderConfig struct {
	Provider addrs.Provider
	Name     string

	// TODO: Figure out how we're going to retain the relevant subset of
	// a provider configuration in the state so that we still have what
	// we need to destroy any associated objects when a provider is removed
	// from the configuration.
	ForEach hcl.Expression

	Config hcl.Body
}

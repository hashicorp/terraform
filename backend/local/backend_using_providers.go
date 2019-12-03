package local

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/providers"
)

var _ backend.UsingProviders = (*Local)(nil)

// RequiredProviders is an implementation of backend.UsingProviders.
//
// The local backend itself does not use any providers, but if it is wrapping
// a state-storage-only backend then it will pass through any providers needed
// by that backend.
func (b *Local) RequiredProviders() []addrs.ProviderType {
	if b.Backend != nil {
		return backend.RequiredProviders(b.Backend)
	}
	return nil
}

// SetProviders is an implementation of backend.UsingProviders.
//
// The local backend itself does not use any providers, but if it is wrapping
// a state-storage-only backend then it would already have passed through
// the wrapped provider's RequiredProviders response, and so it will pass any
// resulting provider factories in as well.
func (b *Local) SetProviders(factories map[addrs.ProviderType]func() providers.Interface) {
	if b.Backend != nil {
		backend.SetProviders(b.Backend, factories)
	}
}

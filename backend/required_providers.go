package backend

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/providers"
)

// RequiredProviders determines the providers required for the given backend.
//
// This is just a covenience wrapper around a type assertion for the optional
// interface backends can implement to indicate provider requirements,
// wrapped so that callers don't need to fuss with such implementation details.
func RequiredProviders(b Backend) []addrs.ProviderType {
	if up, ok := b.(UsingProviders); ok {
		return up.RequiredProviders()
	}
	return nil
}

// SetProviders binds the given backend to the given map of provider factories.
//
// The keys of the map must be exactly the types returned by RequiredProviders,
// or this function may panic or otherwise misbehave.
func SetProviders(b Backend, providers map[addrs.ProviderType]func() providers.Interface) {
	if up, ok := b.(UsingProviders); ok {
		up.SetProviders(providers)
	}
	if len(providers) > 0 {
		panic("SetProviders called with Backend that did not request any providers")
	}
}

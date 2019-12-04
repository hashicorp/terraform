package providers

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plugin/discovery"
)

// Resolver is an interface implemented by objects that are able to resolve
// a given set of resource provider version constraints into Factory
// callbacks.
type Resolver interface {
	// Given a constraint map, return a Factory for each requested provider.
	// If some or all of the constraints cannot be satisfied, return a non-nil
	// slice of errors describing the problems.
	ResolveProviders(reqd discovery.PluginRequirements) (map[addrs.Provider]Factory, []error)
}

// ResolverFunc wraps a callback function and turns it into a Resolver
// implementation, for convenience in situations where a function and its
// associated closure are sufficient as a resolver implementation.
type ResolverFunc func(reqd discovery.PluginRequirements) (map[addrs.Provider]Factory, []error)

// ResolveProviders implements Resolver by calling the
// wrapped function.
func (f ResolverFunc) ResolveProviders(reqd discovery.PluginRequirements) (map[addrs.Provider]Factory, []error) {
	return f(reqd)
}

// ResolverFixed returns a Resolver that has a fixed set of provider factories
// provided by the caller. The returned resolver ignores version constraints
// entirely and just returns the given factory for each requested provider
// name.
//
// This function is primarily used in tests, to provide mock providers or
// in-process providers under test.
func ResolverFixed(factories map[addrs.Provider]Factory) Resolver {
	return ResolverFunc(func(reqd discovery.PluginRequirements) (map[addrs.Provider]Factory, []error) {
		ret := make(map[addrs.Provider]Factory, len(reqd))
		var errs []error
		for name := range reqd {
			fqn := addrs.NewLegacyProvider(name)
			if factory, exists := factories[fqn]; exists {
				ret[fqn] = factory
			} else {
				errs = append(errs, fmt.Errorf("provider %q is not available", name))
			}
		}
		return ret, errs
	})
}

// Factory is a function type that creates a new instance of a resource
// provider, or returns an error if that is impossible.
type Factory func() (Interface, error)

// FactoryFixed is a helper that creates a Factory that just returns some given
// single provider.
//
// Unlike usual factories, the exact same instance is returned for each call
// to the factory and so this must be used in only specialized situations where
// the caller can take care to either not mutate the given provider at all
// or to mutate it in ways that will not cause unexpected behavior for others
// holding the same reference.
func FactoryFixed(p Interface) Factory {
	return func() (Interface, error) {
		return p, nil
	}
}

// ProviderHasResource is a helper that requests schema from the given provider
// and checks if it has a resource type of the given name.
//
// This function is more expensive than it may first appear since it must
// retrieve the entire schema from the underlying provider, and so it should
// be used sparingly and especially not in tight loops.
//
// Since retrieving the provider may fail (e.g. if the provider is accessed
// over an RPC channel that has operational problems), this function will
// return false if the schema cannot be retrieved, under the assumption that
// a subsequent call to do anything with the resource type would fail
// anyway.
func ProviderHasResource(provider Interface, typeName string) bool {
	resp := provider.GetSchema()
	if resp.Diagnostics.HasErrors() {
		return false
	}

	_, exists := resp.ResourceTypes[typeName]
	return exists
}

// ProviderHasDataSource is a helper that requests schema from the given
// provider and checks if it has a data source of the given name.
//
// This function is more expensive than it may first appear since it must
// retrieve the entire schema from the underlying provider, and so it should
// be used sparingly and especially not in tight loops.
//
// Since retrieving the provider may fail (e.g. if the provider is accessed
// over an RPC channel that has operational problems), this function will
// return false if the schema cannot be retrieved, under the assumption that
// a subsequent call to do anything with the data source would fail
// anyway.
func ProviderHasDataSource(provider Interface, dataSourceName string) bool {
	resp := provider.GetSchema()
	if resp.Diagnostics.HasErrors() {
		return false
	}

	_, exists := resp.DataSources[dataSourceName]
	return exists
}

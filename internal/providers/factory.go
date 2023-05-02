// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package providers

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
	resp := provider.GetProviderSchema()
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
	resp := provider.GetProviderSchema()
	if resp.Diagnostics.HasErrors() {
		return false
	}

	_, exists := resp.DataSources[dataSourceName]
	return exists
}

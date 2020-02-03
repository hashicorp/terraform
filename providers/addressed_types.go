package providers

import (
	"sort"

	"github.com/hashicorp/terraform/addrs"
)

// AddressedTypes is a helper that extracts all of the distinct provider
// types from the given list of relative provider configuration addresses.
//
// FIXME: This function is now incorrect, because we can't do a syntax-only
// mapping from a local provider configuration to a provider type. It
// works for now by assuming legacy provider addresses, but will need to be
// replaced by something configuration-aware as part of removing legacy
// provider address reliance.
func AddressedTypes(providerAddrs []addrs.LocalProviderConfig) []addrs.Provider {
	if len(providerAddrs) == 0 {
		return nil
	}
	m := map[string]addrs.Provider{}
	for _, addr := range providerAddrs {
		// FIXME: This will no longer work once we move away from legacy addresses.
		legacyFQN := addrs.NewLegacyProvider(addr.LocalName)
		m[legacyFQN.String()] = legacyFQN
	}

	names := make([]string, 0, len(m))
	for typeName := range m {
		names = append(names, typeName)
	}

	sort.Strings(names) // Stable result for tests

	ret := make([]addrs.Provider, len(names))
	for i, name := range names {
		ret[i] = m[name]
	}

	return ret
}

// AddressedTypesAbs is a helper that extracts all of the distinct provider
// types from the given list of absolute provider configuration addresses.
func AddressedTypesAbs(providerAddrs []addrs.AbsProviderConfig) []addrs.Provider {
	if len(providerAddrs) == 0 {
		return nil
	}
	m := map[string]addrs.Provider{}
	for _, addr := range providerAddrs {
		// FIXME: When changing AbsProviderConfig to include provider FQN,
		// use that directly here instead.
		legacyFQN := addrs.NewLegacyProvider(addr.ProviderConfig.LocalName)
		m[legacyFQN.String()] = legacyFQN
	}

	names := make([]string, 0, len(m))
	for typeName := range m {
		names = append(names, typeName)
	}

	sort.Strings(names) // Stable result for tests

	ret := make([]addrs.Provider, len(names))
	for i, name := range names {
		ret[i] = m[name]
	}

	return ret
}

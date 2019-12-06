package providers

import (
	"sort"

	"github.com/hashicorp/terraform/addrs"
)

// AddressedTypes is a helper that extracts all of the distinct provider
// types from the given list of relative provider configuration addresses.
func AddressedTypes(providerAddrs []addrs.ProviderConfig) []string {
	if len(providerAddrs) == 0 {
		return nil
	}
	m := map[string]struct{}{}
	for _, addr := range providerAddrs {
		m[addr.Type.LegacyString()] = struct{}{}
	}

	names := make([]string, 0, len(m))
	for typeName := range m {
		names = append(names, typeName)
	}

	sort.Strings(names) // Stable result for tests
	return names
}

// AddressedTypesAbs is a helper that extracts all of the distinct provider
// types from the given list of absolute provider configuration addresses.
func AddressedTypesAbs(providerAddrs []addrs.AbsProviderConfig) []string {
	if len(providerAddrs) == 0 {
		return nil
	}
	m := map[string]struct{}{}
	for _, addr := range providerAddrs {
		m[addr.ProviderConfig.Type.LegacyString()] = struct{}{}
	}

	names := make([]string, 0, len(m))
	for typeName := range m {
		names = append(names, typeName)
	}

	sort.Strings(names) // Stable result for tests
	return names
}

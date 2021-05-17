package providers

import (
	"sort"

	"github.com/hashicorp/terraform/addrs"
)

// AddressedTypesAbs is a helper that extracts all of the distinct provider
// types from the given list of absolute provider configuration addresses.
func AddressedTypesAbs(providerAddrs []addrs.AbsProviderConfig) []addrs.Provider {
	if len(providerAddrs) == 0 {
		return nil
	}
	m := map[string]addrs.Provider{}
	for _, addr := range providerAddrs {
		m[addr.Provider.String()] = addr.Provider
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

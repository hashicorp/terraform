package providers

import (
	"testing"

	"github.com/go-test/deep"

	"github.com/hashicorp/terraform/addrs"
)

func TestAddressedTypesAbs(t *testing.T) {
	providerAddrs := []addrs.AbsProviderConfig{
		addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewLegacyProvider("aws"),
		},
		addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewLegacyProvider("aws"),
			Alias:    "foo",
		},
		addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewLegacyProvider("azure"),
		},
		addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewLegacyProvider("null"),
		},
		addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewLegacyProvider("null"),
		},
	}

	got := AddressedTypesAbs(providerAddrs)
	want := []addrs.Provider{
		addrs.NewLegacyProvider("aws"),
		addrs.NewLegacyProvider("azure"),
		addrs.NewLegacyProvider("null"),
	}
	for _, problem := range deep.Equal(got, want) {
		t.Error(problem)
	}
}

package providers

import (
	"testing"

	"github.com/go-test/deep"

	"github.com/hashicorp/terraform/internal/addrs"
)

func TestAddressedTypesAbs(t *testing.T) {
	providerAddrs := []addrs.AbsProviderConfig{
		addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewOfficialProvider("aws"),
		},
		addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewOfficialProvider("aws"),
			Alias:    "foo",
		},
		addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewOfficialProvider("azure"),
		},
		addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewOfficialProvider("null"),
		},
		addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewOfficialProvider("null"),
		},
	}

	got := AddressedTypesAbs(providerAddrs)
	want := []addrs.Provider{
		addrs.NewOfficialProvider("aws"),
		addrs.NewOfficialProvider("azure"),
		addrs.NewOfficialProvider("null"),
	}
	for _, problem := range deep.Equal(got, want) {
		t.Error(problem)
	}
}

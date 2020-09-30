package providers

import (
	"testing"

	"github.com/go-test/deep"

	"github.com/hashicorp/terraform/addrs"
)

func TestAddressedTypesAbs(t *testing.T) {
	providerAddrs := []addrs.AbsProviderConfig{
		{
			Module:   addrs.RootModule,
			Provider: addrs.NewDefaultProvider("aws"),
		},
		{
			Module:   addrs.RootModule,
			Provider: addrs.NewDefaultProvider("aws"),
			Alias:    "foo",
		},
		{
			Module:   addrs.RootModule,
			Provider: addrs.NewDefaultProvider("azure"),
		},
		{
			Module:   addrs.RootModule,
			Provider: addrs.NewDefaultProvider("null"),
		},
		{
			Module:   addrs.RootModule,
			Provider: addrs.NewDefaultProvider("null"),
		},
	}

	got := AddressedTypesAbs(providerAddrs)
	want := []addrs.Provider{
		addrs.NewDefaultProvider("aws"),
		addrs.NewDefaultProvider("azure"),
		addrs.NewDefaultProvider("null"),
	}
	for _, problem := range deep.Equal(got, want) {
		t.Error(problem)
	}
}

package providers

import (
	"testing"

	"github.com/go-test/deep"

	"github.com/hashicorp/terraform/addrs"
)

func TestAddressedTypes(t *testing.T) {
	providerAddrs := []addrs.ProviderConfig{
		{Type: addrs.NewLegacyProvider("aws")},
		{Type: addrs.NewLegacyProvider("aws"), Alias: "foo"},
		{Type: addrs.NewLegacyProvider("azure")},
		{Type: addrs.NewLegacyProvider("null")},
		{Type: addrs.NewLegacyProvider("null")},
	}

	got := AddressedTypes(providerAddrs)
	want := []string{
		"aws",
		"azure",
		"null",
	}
	for _, problem := range deep.Equal(got, want) {
		t.Error(problem)
	}
}

func TestAddressedTypesAbs(t *testing.T) {
	providerAddrs := []addrs.AbsProviderConfig{
		addrs.ProviderConfig{Type: addrs.NewLegacyProvider("aws")}.Absolute(addrs.RootModuleInstance),
		addrs.ProviderConfig{Type: addrs.NewLegacyProvider("aws"), Alias: "foo"}.Absolute(addrs.RootModuleInstance),
		addrs.ProviderConfig{Type: addrs.NewLegacyProvider("azure")}.Absolute(addrs.RootModuleInstance),
		addrs.ProviderConfig{Type: addrs.NewLegacyProvider("null")}.Absolute(addrs.RootModuleInstance),
		addrs.ProviderConfig{Type: addrs.NewLegacyProvider("null")}.Absolute(addrs.RootModuleInstance),
	}

	got := AddressedTypesAbs(providerAddrs)
	want := []string{
		"aws",
		"azure",
		"null",
	}
	for _, problem := range deep.Equal(got, want) {
		t.Error(problem)
	}
}

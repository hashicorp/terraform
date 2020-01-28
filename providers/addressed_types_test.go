package providers

import (
	"testing"

	"github.com/go-test/deep"

	"github.com/hashicorp/terraform/addrs"
)

func TestAddressedTypes(t *testing.T) {
	providerAddrs := []addrs.LocalProviderConfig{
		{LocalType: "aws"},
		{LocalType: "aws", Alias: "foo"},
		{LocalType: "azure"},
		{LocalType: "null"},
		{LocalType: "null"},
	}

	got := AddressedTypes(providerAddrs)
	want := []addrs.Provider{
		addrs.NewLegacyProvider("aws"),
		addrs.NewLegacyProvider("azure"),
		addrs.NewLegacyProvider("null"),
	}
	for _, problem := range deep.Equal(got, want) {
		t.Error(problem)
	}
}

func TestAddressedTypesAbs(t *testing.T) {
	providerAddrs := []addrs.AbsProviderConfig{
		addrs.LocalProviderConfig{LocalType: "aws"}.Absolute(addrs.RootModuleInstance),
		addrs.LocalProviderConfig{LocalType: "aws", Alias: "foo"}.Absolute(addrs.RootModuleInstance),
		addrs.LocalProviderConfig{LocalType: "azure"}.Absolute(addrs.RootModuleInstance),
		addrs.LocalProviderConfig{LocalType: "null"}.Absolute(addrs.RootModuleInstance),
		addrs.LocalProviderConfig{LocalType: "null"}.Absolute(addrs.RootModuleInstance),
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

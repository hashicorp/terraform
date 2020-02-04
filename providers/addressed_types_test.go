package providers

import (
	"testing"

	"github.com/go-test/deep"

	"github.com/hashicorp/terraform/addrs"
)

func TestAddressedTypes(t *testing.T) {
	providerAddrs := []addrs.LocalProviderConfig{
		{LocalName: "aws"},
		{LocalName: "aws", Alias: "foo"},
		{LocalName: "azure"},
		{LocalName: "null"},
		{LocalName: "null"},
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
		addrs.LocalProviderConfig{LocalName: "aws"}.Absolute(addrs.RootModuleInstance),
		addrs.LocalProviderConfig{LocalName: "aws", Alias: "foo"}.Absolute(addrs.RootModuleInstance),
		addrs.LocalProviderConfig{LocalName: "azure"}.Absolute(addrs.RootModuleInstance),
		addrs.LocalProviderConfig{LocalName: "null"}.Absolute(addrs.RootModuleInstance),
		addrs.LocalProviderConfig{LocalName: "null"}.Absolute(addrs.RootModuleInstance),
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

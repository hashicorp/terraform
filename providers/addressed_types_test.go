package providers

import (
	"testing"

	"github.com/go-test/deep"

	"github.com/hashicorp/terraform/addrs"
)

func TestAddressedTypes(t *testing.T) {
	providerAddrs := []addrs.LocalProviderConfig{
		{Type: "aws"},
		{Type: "aws", Alias: "foo"},
		{Type: "azure"},
		{Type: "null"},
		{Type: "null"},
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
		addrs.LocalProviderConfig{Type: "aws"}.Absolute(addrs.RootModuleInstance),
		addrs.LocalProviderConfig{Type: "aws", Alias: "foo"}.Absolute(addrs.RootModuleInstance),
		addrs.LocalProviderConfig{Type: "azure"}.Absolute(addrs.RootModuleInstance),
		addrs.LocalProviderConfig{Type: "null"}.Absolute(addrs.RootModuleInstance),
		addrs.LocalProviderConfig{Type: "null"}.Absolute(addrs.RootModuleInstance),
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

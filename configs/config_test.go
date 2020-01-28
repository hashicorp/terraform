package configs

import (
	"testing"

	"github.com/go-test/deep"

	"github.com/hashicorp/terraform/addrs"
)

func TestConfigProviderTypes(t *testing.T) {
	mod, diags := testModuleFromFile("testdata/valid-files/providers-explicit-implied.tf")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	cfg, diags := BuildConfig(mod, nil)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	got := cfg.ProviderTypes()
	want := []addrs.Provider{
		addrs.NewLegacyProvider("aws"),
		addrs.NewLegacyProvider("null"),
		addrs.NewLegacyProvider("template"),
	}
	for _, problem := range deep.Equal(got, want) {
		t.Error(problem)
	}
}

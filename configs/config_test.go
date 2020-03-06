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

func TestConfigResolveAbsProviderAddr(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/providers-explicit-fqn")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	cfg, diags := BuildConfig(mod, nil)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	t.Run("already absolute", func(t *testing.T) {
		addr := addrs.AbsProviderConfig{
			Module:   addrs.RootModuleInstance,
			Provider: addrs.NewLegacyProvider("test"),
			Alias:    "boop",
		}
		got := cfg.ResolveAbsProviderAddr(addr, addrs.RootModuleInstance)
		if got, want := got.String(), addr.String(); got != want {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("local, implied mapping", func(t *testing.T) {
		addr := addrs.LocalProviderConfig{
			LocalName: "implied",
			Alias:     "boop",
		}
		got := cfg.ResolveAbsProviderAddr(addr, addrs.RootModuleInstance)
		want := addrs.AbsProviderConfig{
			Module: addrs.RootModuleInstance,
			// FIXME: At the time of writing we still have LocalProviderConfig
			// nested inside AbsProviderConfig, but a future change will
			// stop tis embedding and just have an addrs.Provider and an alias
			// string here, at which point the correct result will be:
			//    Provider as the addrs repr of "registry.terraform.io/hashicorp/implied"
			//    Alias as "boop".
			Provider: addrs.NewLegacyProvider("implied"),
			Alias:    "boop",
		}
		if got, want := got.String(), want.String(); got != want {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("local, explicit mapping", func(t *testing.T) {
		addr := addrs.LocalProviderConfig{
			LocalName: "foo-test", // this is explicitly set in the config
			Alias:     "boop",
		}
		got := cfg.ResolveAbsProviderAddr(addr, addrs.RootModuleInstance)
		want := addrs.AbsProviderConfig{
			Module:   addrs.RootModuleInstance,
			Provider: addrs.NewProvider(addrs.DefaultRegistryHost, "foo", "test"),
			Alias:    "boop",
		}
		if got, want := got.String(), want.String(); got != want {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
		}
	})
}

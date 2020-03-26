package configs

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/google/go-cmp/cmp"

	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
)

func TestConfigProviderTypes(t *testing.T) {
	// nil cfg should return an empty map
	got := NewEmptyConfig().ProviderTypes()
	if len(got) != 0 {
		t.Fatal("expected empty result from empty config")
	}

	cfg, diags := testModuleConfigFromFile("testdata/valid-files/providers-explicit-implied.tf")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	got = cfg.ProviderTypes()
	want := []addrs.Provider{
		addrs.NewLegacyProvider("aws"),
		addrs.NewLegacyProvider("null"),
		addrs.NewLegacyProvider("template"),
	}
	for _, problem := range deep.Equal(got, want) {
		t.Error(problem)
	}
}

func TestConfigProviderTypes_nested(t *testing.T) {
	// basic test with a nil config
	c := NewEmptyConfig()
	got := c.ProviderTypes()
	if len(got) != 0 {
		t.Fatalf("wrong result!\ngot: %#v\nwant: nil\n", got)
	}

	// config with two provider sources, and one implicit (default) provider
	cfg, diags := testNestedModuleConfigFromDir(t, "testdata/valid-modules/nested-providers-fqns")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	got = cfg.ProviderTypes()
	want := []addrs.Provider{
		// FIXME: this will be updated to NewDefaultProvider as we remove `Legacy*`
		addrs.NewLegacyProvider("test"),
		addrs.NewProvider(addrs.DefaultRegistryHost, "bar", "test"),
		addrs.NewProvider(addrs.DefaultRegistryHost, "foo", "test"),
	}

	for _, problem := range deep.Equal(got, want) {
		t.Error(problem)
	}
}

func TestConfigResolveAbsProviderAddr(t *testing.T) {
	cfg, diags := testModuleConfigFromDir("testdata/providers-explicit-fqn")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	t.Run("already absolute", func(t *testing.T) {
		addr := addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewLegacyProvider("test"),
			Alias:    "boop",
		}
		got := cfg.ResolveAbsProviderAddr(addr, addrs.RootModule)
		if got, want := got.String(), addr.String(); got != want {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("local, implied mapping", func(t *testing.T) {
		addr := addrs.LocalProviderConfig{
			LocalName: "implied",
			Alias:     "boop",
		}
		got := cfg.ResolveAbsProviderAddr(addr, addrs.RootModule)
		want := addrs.AbsProviderConfig{
			Module: addrs.RootModule,
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
		got := cfg.ResolveAbsProviderAddr(addr, addrs.RootModule)
		want := addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewProvider(addrs.DefaultRegistryHost, "foo", "test"),
			Alias:    "boop",
		}
		if got, want := got.String(), want.String(); got != want {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestConfigProviderRequirements(t *testing.T) {
	cfg, diags := testNestedModuleConfigFromDir(t, "testdata/provider-reqs")
	assertNoDiagnostics(t, diags)

	tlsProvider := addrs.NewProvider(
		addrs.DefaultRegistryHost,
		"hashicorp", "tls",
	)
	happycloudProvider := addrs.NewProvider(
		svchost.Hostname("tf.example.com"),
		"awesomecorp", "happycloud",
	)
	// FIXME: these two are legacy ones for now because the config loader
	// isn't using default configurations fully yet.
	// Once that changes, these should be default-shaped ones like tlsProvider
	// above.
	nullProvider := addrs.NewLegacyProvider("null")
	randomProvider := addrs.NewLegacyProvider("random")
	impliedProvider := addrs.NewLegacyProvider("implied")

	got, diags := cfg.ProviderRequirements()
	assertNoDiagnostics(t, diags)
	want := getproviders.Requirements{
		// the nullProvider constraints from the two modules are merged
		nullProvider:       getproviders.MustParseVersionConstraints("~> 2.0.0, 2.0.1"),
		randomProvider:     getproviders.MustParseVersionConstraints("~> 1.2.0"),
		tlsProvider:        getproviders.MustParseVersionConstraints("~> 3.0"),
		impliedProvider:    nil,
		happycloudProvider: nil,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestProviderForConfigAddr(t *testing.T) {
	cfg, diags := testModuleConfigFromDir("testdata/valid-modules/providers-fqns")
	assertNoDiagnostics(t, diags)

	got := cfg.ProviderForConfigAddr(addrs.NewDefaultLocalProviderConfig("foo-test"))
	want := addrs.NewProvider(addrs.DefaultRegistryHost, "foo", "test")
	if !got.Equals(want) {
		t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
	}

	// now check a provider that isn't in the configuration. It should return a NewLegacyProvider.
	got = cfg.ProviderForConfigAddr(addrs.NewDefaultLocalProviderConfig("bar-test"))
	want = addrs.NewLegacyProvider("bar-test")
	if !got.Equals(want) {
		t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
	}
}

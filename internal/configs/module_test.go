package configs

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
)

// TestNewModule_provider_fqns exercises module.gatherProviderLocalNames()
func TestNewModule_provider_local_name(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/providers-explicit-fqn")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	p := addrs.NewProvider(addrs.DefaultProviderRegistryHost, "foo", "test")
	if name, exists := mod.ProviderLocalNames[p]; !exists {
		t.Fatal("provider FQN foo/test not found")
	} else {
		if name != "foo-test" {
			t.Fatalf("provider localname mismatch: got %s, want foo-test", name)
		}
	}

	// ensure the reverse lookup (fqn to local name) works as well
	localName := mod.LocalNameForProvider(p)
	if localName != "foo-test" {
		t.Fatal("provider local name not found")
	}

	// if there is not a local name for a provider, it should return the type name
	localName = mod.LocalNameForProvider(addrs.NewDefaultProvider("nonexist"))
	if localName != "nonexist" {
		t.Error("wrong local name returned for a non-local provider")
	}

	// can also look up the "terraform" provider and see that it sources is
	// allowed to be overridden, even though there is a builtin provider
	// called "terraform".
	p = addrs.NewProvider(addrs.DefaultProviderRegistryHost, "not-builtin", "not-terraform")
	if name, exists := mod.ProviderLocalNames[p]; !exists {
		t.Fatal("provider FQN not-builtin/not-terraform not found")
	} else {
		if name != "terraform" {
			t.Fatalf("provider localname mismatch: got %s, want terraform", name)
		}
	}
}

// This test validates the provider FQNs set in each Resource
func TestNewModule_resource_providers(t *testing.T) {
	cfg, diags := testNestedModuleConfigFromDir(t, "testdata/valid-modules/nested-providers-fqns")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	// both the root and child module have two resources, one which should use
	// the default implied provider and one explicitly using a provider set in
	// required_providers
	wantImplicit := addrs.NewDefaultProvider("test")
	wantFoo := addrs.NewProvider(addrs.DefaultProviderRegistryHost, "foo", "test")
	wantBar := addrs.NewProvider(addrs.DefaultProviderRegistryHost, "bar", "test")

	// root module
	if !cfg.Module.ManagedResources["test_instance.explicit"].Provider.Equals(wantFoo) {
		t.Fatalf("wrong provider for \"test_instance.explicit\"\ngot:  %s\nwant: %s",
			cfg.Module.ManagedResources["test_instance.explicit"].Provider,
			wantFoo,
		)
	}
	if !cfg.Module.ManagedResources["test_instance.implicit"].Provider.Equals(wantImplicit) {
		t.Fatalf("wrong provider for \"test_instance.implicit\"\ngot:  %s\nwant: %s",
			cfg.Module.ManagedResources["test_instance.implicit"].Provider,
			wantImplicit,
		)
	}

	// a data source
	if !cfg.Module.DataResources["data.test_resource.explicit"].Provider.Equals(wantFoo) {
		t.Fatalf("wrong provider for \"module.child.test_instance.explicit\"\ngot:  %s\nwant: %s",
			cfg.Module.ManagedResources["test_instance.explicit"].Provider,
			wantBar,
		)
	}

	// child module
	cm := cfg.Children["child"].Module
	if !cm.ManagedResources["test_instance.explicit"].Provider.Equals(wantBar) {
		t.Fatalf("wrong provider for \"module.child.test_instance.explicit\"\ngot:  %s\nwant: %s",
			cfg.Module.ManagedResources["test_instance.explicit"].Provider,
			wantBar,
		)
	}
	if !cm.ManagedResources["test_instance.implicit"].Provider.Equals(wantImplicit) {
		t.Fatalf("wrong provider for \"module.child.test_instance.implicit\"\ngot:  %s\nwant: %s",
			cfg.Module.ManagedResources["test_instance.implicit"].Provider,
			wantImplicit,
		)
	}
}

func TestProviderForLocalConfig(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/providers-explicit-fqn")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}
	lc := addrs.LocalProviderConfig{LocalName: "foo-test"}
	got := mod.ProviderForLocalConfig(lc)
	want := addrs.NewProvider(addrs.DefaultProviderRegistryHost, "foo", "test")
	if !got.Equals(want) {
		t.Fatalf("wrong result! got %#v, want %#v\n", got, want)
	}
}

// At most one required_providers block per module is permitted.
func TestModule_required_providers_multiple(t *testing.T) {
	_, diags := testModuleFromDir("testdata/invalid-modules/multiple-required-providers")
	if !diags.HasErrors() {
		t.Fatal("module should have error diags, but does not")
	}

	want := `Duplicate required providers configuration`
	if got := diags.Error(); !strings.Contains(got, want) {
		t.Fatalf("expected error to contain %q\nerror was:\n%s", want, got)
	}
}

// A module may have required_providers configured in files loaded later than
// resources. These provider settings should still be reflected in the
// resources' configuration.
func TestModule_required_providers_after_resource(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/valid-modules/required-providers-after-resource")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	want := addrs.NewProvider(addrs.DefaultProviderRegistryHost, "foo", "test")

	req, exists := mod.ProviderRequirements.RequiredProviders["test"]
	if !exists {
		t.Fatal("no provider requirements found for \"test\"")
	}
	if req.Type != want {
		t.Errorf("wrong provider addr for \"test\"\ngot:  %s\nwant: %s",
			req.Type, want,
		)
	}

	if got := mod.ManagedResources["test_instance.my-instance"].Provider; !got.Equals(want) {
		t.Errorf("wrong provider addr for \"test_instance.my-instance\"\ngot:  %s\nwant: %s",
			got, want,
		)
	}
}

// We support overrides for required_providers blocks, which should replace the
// entire block for each provider localname, leaving other blocks unaffected.
// This should also be reflected in any resources in the module using this
// provider.
func TestModule_required_provider_overrides(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/valid-modules/required-providers-overrides")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	// The foo provider and resource should be unaffected
	want := addrs.NewProvider(addrs.DefaultProviderRegistryHost, "acme", "foo")
	req, exists := mod.ProviderRequirements.RequiredProviders["foo"]
	if !exists {
		t.Fatal("no provider requirements found for \"foo\"")
	}
	if req.Type != want {
		t.Errorf("wrong provider addr for \"foo\"\ngot:  %s\nwant: %s",
			req.Type, want,
		)
	}
	if got := mod.ManagedResources["foo_thing.ft"].Provider; !got.Equals(want) {
		t.Errorf("wrong provider addr for \"foo_thing.ft\"\ngot:  %s\nwant: %s",
			got, want,
		)
	}

	// The bar provider and resource should be using the override config
	want = addrs.NewProvider(addrs.DefaultProviderRegistryHost, "blorp", "bar")
	req, exists = mod.ProviderRequirements.RequiredProviders["bar"]
	if !exists {
		t.Fatal("no provider requirements found for \"bar\"")
	}
	if req.Type != want {
		t.Errorf("wrong provider addr for \"bar\"\ngot:  %s\nwant: %s",
			req.Type, want,
		)
	}
	if gotVer, wantVer := req.Requirement.Required.String(), "~>2.0.0"; gotVer != wantVer {
		t.Errorf("wrong provider version constraint for \"bar\"\ngot:  %s\nwant: %s",
			gotVer, wantVer,
		)
	}
	if got := mod.ManagedResources["bar_thing.bt"].Provider; !got.Equals(want) {
		t.Errorf("wrong provider addr for \"bar_thing.bt\"\ngot:  %s\nwant: %s",
			got, want,
		)
	}
}

// Resources without explicit provider configuration are assigned a provider
// implied based on the resource type. For example, this resource:
//
//  resource foo_instance "test" { }
//
// is assigned a provider with type "foo".
//
// To find the correct provider, we first look in the module's provider
// requirements map for a local name matching the resource type, and fall back
// to a default provider if none is found. This applies to both managed and
// data resources.
func TestModule_implied_provider(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/valid-modules/implied-providers")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	// The three providers used in the config resources
	foo := addrs.NewProvider("registry.acme.corp", "acme", "foo")
	whatever := addrs.NewProvider(addrs.DefaultProviderRegistryHost, "acme", "something")
	bar := addrs.NewDefaultProvider("bar")

	// Verify that the registry.acme.corp/acme/foo provider is defined in the
	// module provider requirements with local name "foo"
	req, exists := mod.ProviderRequirements.RequiredProviders["foo"]
	if !exists {
		t.Fatal("no provider requirements found for \"foo\"")
	}
	if req.Type != foo {
		t.Errorf("wrong provider addr for \"foo\"\ngot:  %s\nwant: %s",
			req.Type, foo,
		)
	}

	// Verify that the acme/something provider is defined in the
	// module provider requirements with local name "whatever"
	req, exists = mod.ProviderRequirements.RequiredProviders["whatever"]
	if !exists {
		t.Fatal("no provider requirements found for \"foo\"")
	}
	if req.Type != whatever {
		t.Errorf("wrong provider addr for \"whatever\"\ngot:  %s\nwant: %s",
			req.Type, whatever,
		)
	}

	// Check that resources are assigned the correct providers: foo_* resources
	// should have the custom foo provider, bar_* resources the default bar
	// provider.
	tests := []struct {
		Address  string
		Provider addrs.Provider
	}{
		{"foo_resource.a", foo},
		{"data.foo_resource.b", foo},
		{"bar_resource.c", bar},
		{"data.bar_resource.d", bar},
		{"whatever_resource.e", whatever},
		{"data.whatever_resource.f", whatever},
	}
	for _, test := range tests {
		resources := mod.ManagedResources
		if strings.HasPrefix(test.Address, "data.") {
			resources = mod.DataResources
		}
		resource, exists := resources[test.Address]
		if !exists {
			t.Errorf("could not find resource %q in %#v", test.Address, resources)
			continue
		}
		if got := resource.Provider; !got.Equals(test.Provider) {
			t.Errorf("wrong provider addr for %q\ngot:  %s\nwant: %s",
				test.Address, got, test.Provider,
			)
		}
	}
}

func TestImpliedProviderForUnqualifiedType(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/valid-modules/implied-providers")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	foo := addrs.NewProvider("registry.acme.corp", "acme", "foo")
	whatever := addrs.NewProvider(addrs.DefaultProviderRegistryHost, "acme", "something")
	bar := addrs.NewDefaultProvider("bar")
	tf := addrs.NewBuiltInProvider("terraform")

	tests := []struct {
		Type     string
		Provider addrs.Provider
	}{
		{"foo", foo},
		{"whatever", whatever},
		{"bar", bar},
		{"terraform", tf},
	}
	for _, test := range tests {
		got := mod.ImpliedProviderForUnqualifiedType(test.Type)
		if !got.Equals(test.Provider) {
			t.Errorf("wrong result for %q: got %#v, want %#v\n", test.Type, got, test.Provider)
		}
	}
}

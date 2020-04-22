package configs

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

// TestNewModule_provider_fqns exercises module.gatherProviderLocalNames()
func TestNewModule_provider_local_name(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/providers-explicit-fqn")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	p := addrs.NewProvider(addrs.DefaultRegistryHost, "foo", "test")
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
	localName = mod.LocalNameForProvider(addrs.NewLegacyProvider("nonexist"))
	if localName != "nonexist" {
		t.Error("wrong local name returned for a non-local provider")
	}

	// can also look up the "terraform" provider and see that it sources is
	// allowed to be overridden, even though there is a builtin provider
	// called "terraform".
	p = addrs.NewProvider(addrs.DefaultRegistryHost, "not-builtin", "not-terraform")
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
	wantFoo := addrs.NewProvider(addrs.DefaultRegistryHost, "foo", "test")
	wantBar := addrs.NewProvider(addrs.DefaultRegistryHost, "bar", "test")

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

func TestModule_required_providers_multiple(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/valid-modules/multiple-required-providers")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	want := addrs.NewProvider(addrs.DefaultRegistryHost, "foo", "test")

	req, exists := mod.ProviderRequirements["test"]
	if !exists {
		t.Fatal("no provider requirements found for \"test\"")
	}
	if req.Type != want {
		t.Errorf("wrong provider addr for %q\ngot:  %s\nwant: %s",
			"test", req.Type, want,
		)
	}
}

func TestModule_required_providers_after_resource(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/valid-modules/required-providers-after-resource")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	want := addrs.NewProvider(addrs.DefaultRegistryHost, "foo", "test")

	req, exists := mod.ProviderRequirements["test"]
	if !exists {
		t.Fatal("no provider requirements found for \"test\"")
	}
	if req.Type != want {
		t.Errorf("wrong provider addr for %q\ngot:  %s\nwant: %s",
			"test", req.Type, want,
		)
	}

	if got := mod.ManagedResources["test_instance.my-instance"].Provider; !got.Equals(want) {
		t.Errorf("wrong provider addr for %q\ngot:  %s\nwant: %s",
			"test_instance.my-instance", got, want,
		)
	}
}

func TestModule_required_providers_conflicting_sources(t *testing.T) {
	_, diags := testModuleFromDir("testdata/invalid-modules/conflicting-required-providers")
	if !diags.HasErrors() {
		t.Fatal("module should have error diags, but does not")
	}

	want := `Multiple provider sources specified for "test": "registry.terraform.io/acme/test", "registry.terraform.io/foo/test"`
	if got := diags.Error(); !strings.Contains(got, want) {
		t.Fatalf("expected error to contain %q\nerror was:\n%s", want, got)
	}
}

func TestProviderForLocalConfig(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/providers-explicit-fqn")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}
	lc := addrs.LocalProviderConfig{LocalName: "foo-test"}
	got := mod.ProviderForLocalConfig(lc)
	want := addrs.NewProvider(addrs.DefaultRegistryHost, "foo", "test")
	if !got.Equals(want) {
		t.Fatalf("wrong result! got %#v, want %#v\n", got, want)
	}
}

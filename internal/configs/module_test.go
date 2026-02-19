// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
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
//	resource "foo_instance" "test" {}
//
// ...is assigned to whichever provider has local name "foo" in the current
// module.
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

func TestModule_backend_overrides_a_backend(t *testing.T) {
	t.Run("it can override a backend block with a different backend block", func(t *testing.T) {
		mod, diags := testModuleFromDir("testdata/valid-modules/override-backend")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		gotType := mod.Backend.Type
		wantType := "bar"

		if gotType != wantType {
			t.Errorf("wrong result for backend type: got %#v, want %#v\n", gotType, wantType)
		}

		attrs, _ := mod.Backend.Config.JustAttributes()

		gotAttr, diags := attrs["path"].Expr.Value(nil)
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		wantAttr := cty.StringVal("CHANGED/relative/path/to/terraform.tfstate")

		if !gotAttr.RawEquals(wantAttr) {
			t.Errorf("wrong result for backend 'path': got %#v, want %#v\n", gotAttr, wantAttr)
		}
	})
}

// Unlike most other overrides, backend blocks do not require a base configuration in a primary
// configuration file, as an omitted backend there implies the local backend.
func TestModule_backend_overrides_no_base(t *testing.T) {
	t.Run("it can introduce a backend block via overrides when the base config has has no cloud or backend blocks", func(t *testing.T) {
		mod, diags := testModuleFromDir("testdata/valid-modules/override-backend-no-base")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		if mod.Backend == nil {
			t.Errorf("expected module Backend not to be nil")
		}
	})
}

func TestModule_cloud_overrides_a_backend(t *testing.T) {
	t.Run("it can override a backend block with a cloud block", func(t *testing.T) {
		mod, diags := testModuleFromDir("testdata/valid-modules/override-backend-with-cloud")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		if mod.Backend != nil {
			t.Errorf("expected module Backend to be nil")
		}

		if mod.CloudConfig == nil {
			t.Errorf("expected module CloudConfig not to be nil")
		}
	})
}

func TestModule_cloud_overrides_cloud(t *testing.T) {
	t.Run("it can override a cloud block with a different cloud block", func(t *testing.T) {
		mod, diags := testModuleFromDir("testdata/valid-modules/override-cloud")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		attrs, _ := mod.CloudConfig.Config.JustAttributes()

		gotAttr, diags := attrs["organization"].Expr.Value(nil)
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		wantAttr := cty.StringVal("CHANGED")

		if !gotAttr.RawEquals(wantAttr) {
			t.Errorf("wrong result for Cloud 'organization': got %#v, want %#v\n", gotAttr, wantAttr)
		}

		// The override should have completely replaced the cloud block in the primary file, no merging
		if attrs["should_not_be_present_with_override"] != nil {
			t.Errorf("expected 'should_not_be_present_with_override' attribute to be nil")
		}
	})
}

// Unlike most other overrides, cloud blocks do not require a base configuration in a primary
// configuration file, as an omitted backend there implies the local backend and cloud blocks
// override backends.
func TestModule_cloud_overrides_no_base(t *testing.T) {
	t.Run("it can introduce a cloud block via overrides when the base config has no cloud or backend blocks", func(t *testing.T) {

		mod, diags := testModuleFromDir("testdata/valid-modules/override-cloud-no-base")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		if mod.CloudConfig == nil {
			t.Errorf("expected module CloudConfig not to be nil")
		}
	})
}

func TestModule_backend_overrides_cloud(t *testing.T) {
	t.Run("it can override a cloud block with a backend block", func(t *testing.T) {
		mod, diags := testModuleFromDir("testdata/valid-modules/override-cloud-with-backend")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		gotType := mod.Backend.Type
		wantType := "override"

		if gotType != wantType {
			t.Errorf("wrong result for backend type: got %#v, want %#v\n", gotType, wantType)
		}

		attrs, _ := mod.Backend.Config.JustAttributes()

		gotAttr, diags := attrs["path"].Expr.Value(nil)
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		wantAttr := cty.StringVal("value from override")

		if !gotAttr.RawEquals(wantAttr) {
			t.Errorf("wrong result for backend 'path': got %#v, want %#v\n", gotAttr, wantAttr)
		}
	})
}

func TestModule_cloud_duplicate_overrides(t *testing.T) {
	t.Run("it raises an error when a override file contains multiple cloud blocks", func(t *testing.T) {
		_, diags := testModuleFromDir("testdata/invalid-modules/override-cloud-duplicates")
		want := `Duplicate HCP Terraform configurations`
		if got := diags.Error(); !strings.Contains(got, want) {
			t.Fatalf("expected module error to contain %q\nerror was:\n%s", want, got)
		}
	})
}

func TestModule_backend_multiple(t *testing.T) {
	t.Run("it detects when two backend blocks are present within the same module in separate files", func(t *testing.T) {
		_, diags := testModuleFromDir("testdata/invalid-modules/multiple-backends")
		if !diags.HasErrors() {
			t.Fatal("module should have error diags, but does not")
		}

		want := `Duplicate 'backend' configuration block`
		if got := diags.Error(); !strings.Contains(got, want) {
			t.Fatalf("expected error to contain %q\nerror was:\n%s", want, got)
		}
	})
}

func TestModule_cloud_multiple(t *testing.T) {
	t.Run("it detects when two cloud blocks are present within the same module in separate files", func(t *testing.T) {

		_, diags := testModuleFromDir("testdata/invalid-modules/multiple-cloud")
		if !diags.HasErrors() {
			t.Fatal("module should have error diags, but does not")
		}

		want := `Duplicate HCP Terraform configurations`
		if got := diags.Error(); !strings.Contains(got, want) {
			t.Fatalf("expected error to contain %q\nerror was:\n%s", want, got)
		}
	})
}

// Cannot combine use of backend, cloud, state_store blocks.
func TestModule_conflicting_backend_cloud_stateStore(t *testing.T) {
	testCases := map[string]struct {
		dir              string
		wantMsg          string
		allowExperiments bool
	}{
		"cloud backends conflict": {
			// detects when both cloud and backend blocks are in the same terraform block
			dir:     "testdata/invalid-modules/conflict-cloud-backend",
			wantMsg: `Conflicting 'cloud' and 'backend' configuration blocks are present`,
		},
		"cloud backends conflict separate": {
			// it detects when both cloud and backend blocks are present in the same module in separate files
			dir:     "testdata/invalid-modules/conflict-cloud-backend-separate-files",
			wantMsg: `Conflicting 'cloud' and 'backend' configuration blocks are present`,
		},
		"cloud state store conflict": {
			// detects when both cloud and state_store blocks are in the same terraform block
			dir:              "testdata/invalid-modules/conflict-cloud-statestore",
			wantMsg:          `Conflicting 'cloud' and 'state_store' configuration blocks are present`,
			allowExperiments: true,
		},
		"cloud state store conflict separate": {
			// it detects when both cloud and state_store blocks are present in the same module in separate files
			dir:              "testdata/invalid-modules/conflict-cloud-statestore-separate-files",
			wantMsg:          `Conflicting 'cloud' and 'state_store' configuration blocks are present`,
			allowExperiments: true,
		},
		"state store backend conflict": {
			// it detects when both state_store and backend blocks are in the same terraform block
			dir:              "testdata/invalid-modules/conflict-statestore-backend",
			wantMsg:          `Conflicting 'state_store' and 'backend' configuration blocks are present`,
			allowExperiments: true,
		},
		"state store backend conflict separate": {
			// it detects when both state_store and backend blocks are present in the same module in separate files
			dir:              "testdata/invalid-modules/conflict-statestore-backend-separate-files",
			wantMsg:          `Conflicting 'state_store' and 'backend' configuration blocks are present`,
			allowExperiments: true,
		},
		"cloud backend state store conflict": {
			// it detects all 3 of cloud, state_storage and backend blocks are in the same terraform block
			dir:              "testdata/invalid-modules/conflict-cloud-backend-statestore",
			wantMsg:          `Only one of 'cloud', 'state_store', or 'backend' configuration blocks are allowed`,
			allowExperiments: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.dir, func(t *testing.T) {
			var diags hcl.Diagnostics
			if tc.allowExperiments {
				// TODO(SarahFrench/radeksimko) - disable experiments in this test once the feature is GA.
				_, diags = testModuleFromDirWithExperiments(tc.dir)
			} else {
				_, diags = testModuleFromDir(tc.dir)
			}
			if !diags.HasErrors() {
				t.Fatal("module should have error diags, but does not")
			}

			if got := diags.Error(); !strings.Contains(got, tc.wantMsg) {
				t.Fatalf("expected error to contain %q\nerror was:\n%s", tc.wantMsg, got)
			}
		})
	}
}

func TestModule_stateStore_overrides_stateStore(t *testing.T) {
	t.Run("it can override a state_store block with a different state_store block", func(t *testing.T) {
		// TODO(SarahFrench/radeksimko) - disable experiments in this test once the feature is GA.
		mod, diags := testModuleFromDirWithExperiments("testdata/valid-modules/override-state-store")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		if mod.StateStore == nil {
			t.Fatal("expected parsed module to include a state store, found none")
		}

		// Check type override
		gotType := mod.StateStore.Type
		wantType := "foo_override"
		if gotType != wantType {
			t.Errorf("wrong result for state_store type: got %#v, want %#v\n", gotType, wantType)
		}

		// Check custom attribute override
		attrs, _ := mod.StateStore.Config.JustAttributes()
		gotAttr, diags := attrs["custom_attr"].Expr.Value(nil)
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}
		wantAttr := cty.StringVal("override")
		if !gotAttr.RawEquals(wantAttr) {
			t.Errorf("wrong result for state_store 'custom_attr': got %#v, want %#v\n", gotAttr, wantAttr)
		}

		// Check provider reference override
		wantLocalName := "bar"
		if mod.StateStore.Provider.Name != wantLocalName {
			t.Errorf("wrong result for state_store 'provider' value's local name: got %#v, want %#v\n", mod.StateStore.Provider.Name, wantLocalName)
		}
	})
}

// Unlike most other overrides, state_store blocks do not require a base configuration in a primary
// configuration file, as an omitted backend there implies the local backend.
func TestModule_stateStore_override_no_base(t *testing.T) {
	t.Run("it can introduce a state_store block via overrides when the base config has has no cloud, backend, or state_store blocks", func(t *testing.T) {
		// TODO(SarahFrench/radeksimko) - disable experiments in this test once the feature is GA.
		mod, diags := testModuleFromDirWithExperiments("testdata/valid-modules/override-state-store-no-base")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		if mod.StateStore == nil {
			t.Errorf("expected module StateStore not to be nil")
		}
	})
}

func TestModule_stateStore_overrides_backend(t *testing.T) {
	t.Run("it can override a backend block with a state_store block", func(t *testing.T) {
		// TODO(SarahFrench/radeksimko) - disable experiments in this test once the feature is GA.
		mod, diags := testModuleFromDirWithExperiments("testdata/valid-modules/override-backend-with-state-store")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		// Backend not set
		if mod.Backend != nil {
			t.Errorf("backend should not be set: got %#v\n", mod.Backend)
		}

		// Check state_store
		if mod.StateStore == nil {
			t.Fatal("expected parsed module to include a state store, found none")
		}

		gotType := mod.StateStore.Type
		wantType := "foo_override"
		if gotType != wantType {
			t.Errorf("wrong result for state_store type: got %#v, want %#v\n", gotType, wantType)
		}

		// Not necessary to assert all values in state_store
	})
}

func TestModule_stateStore_overrides_cloud(t *testing.T) {
	t.Run("it can override a cloud block with a state_store block", func(t *testing.T) {
		// TODO(SarahFrench/radeksimko) - disable experiments in this test once the feature is GA.
		mod, diags := testModuleFromDirWithExperiments("testdata/valid-modules/override-cloud-with-state-store")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		// CloudConfig not set
		if mod.CloudConfig != nil {
			t.Errorf("backend should not be set: got %#v\n", mod.Backend)
		}

		// Check state_store
		if mod.StateStore == nil {
			t.Fatal("expected parsed module to include a state store, found none")
		}
		gotType := mod.StateStore.Type
		wantType := "foo_override"
		if gotType != wantType {
			t.Errorf("wrong result for state_store type: got %#v, want %#v\n", gotType, wantType)
		}

		// Not necessary to assert all values in state_store
	})
}

func TestModule_state_store_multiple(t *testing.T) {
	t.Run("it detects when two state_store blocks are present within the same module in separate files", func(t *testing.T) {
		// TODO(SarahFrench/radeksimko) - disable experiments in this test once the feature is GA.
		_, diags := testModuleFromDirWithExperiments("testdata/invalid-modules/multiple-state-store")
		if !diags.HasErrors() {
			t.Fatal("module should have error diags, but does not")
		}

		want := `Duplicate 'state_store' configuration block`
		if got := diags.Error(); !strings.Contains(got, want) {
			t.Fatalf("expected error to contain %q\nerror was:\n%s", want, got)
		}
	})
}

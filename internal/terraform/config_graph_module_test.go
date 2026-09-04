// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

func TestModule_nested_import_blocks(t *testing.T) {
	m, diags := testNestedModuleConfigFromDir(t, "testdata/config-graph/valid-modules/import-blocks-in-module")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	if len(m.Children["child"].Module.Import) != 2 {
		t.Fatal("child module is missing nested import blocks")
	}
}

// This test validates the provider FQNs set in each Resource
func TestNewModule_resource_providers(t *testing.T) {
	cfg, diags := testNestedModuleConfigFromDir(t, "testdata/config-graph/valid-modules/nested-providers-fqns")
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

// testNestedModuleConfigFromDir reads configuration from the given directory path as
// a module with (optional) submodules and returns its configuration. This is a
// helper for use in unit tests.
func testNestedModuleConfigFromDir(t *testing.T, path string) (*configs.Config, hcl.Diagnostics) {
	t.Helper()

	parser := configs.NewParser(nil)
	mod, diags := parser.LoadConfigDir(path)
	if mod == nil {
		t.Fatal("got nil root module; want non-nil")
	}

	cfg, nestedDiags := buildNestedModuleConfig(mod, path, parser)

	diags = append(diags, nestedDiags...)
	return cfg, diags
}

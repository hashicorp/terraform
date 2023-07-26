// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	svchost "github.com/hashicorp/terraform-svchost"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
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
		addrs.NewDefaultProvider("aws"),
		addrs.NewDefaultProvider("local"),
		addrs.NewDefaultProvider("null"),
		addrs.NewDefaultProvider("template"),
		addrs.NewDefaultProvider("test"),
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
		addrs.NewProvider(addrs.DefaultProviderRegistryHost, "bar", "test"),
		addrs.NewProvider(addrs.DefaultProviderRegistryHost, "foo", "test"),
		addrs.NewDefaultProvider("test"),
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
			Provider: addrs.NewDefaultProvider("test"),
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
			Module:   addrs.RootModule,
			Provider: addrs.NewDefaultProvider("implied"),
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
			Provider: addrs.NewProvider(addrs.DefaultProviderRegistryHost, "foo", "test"),
			Alias:    "boop",
		}
		if got, want := got.String(), want.String(); got != want {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestConfigProviderRequirements(t *testing.T) {
	cfg, diags := testNestedModuleConfigFromDir(t, "testdata/provider-reqs")
	// TODO: Version Constraint Deprecation.
	// Once we've removed the version argument from provider configuration
	// blocks, this can go back to expected 0 diagnostics.
	// assertNoDiagnostics(t, diags)
	assertDiagnosticCount(t, diags, 1)
	assertDiagnosticSummary(t, diags, "Version constraints inside provider configuration blocks are deprecated")

	tlsProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost,
		"hashicorp", "tls",
	)
	happycloudProvider := addrs.NewProvider(
		svchost.Hostname("tf.example.com"),
		"awesomecorp", "happycloud",
	)
	nullProvider := addrs.NewDefaultProvider("null")
	randomProvider := addrs.NewDefaultProvider("random")
	impliedProvider := addrs.NewDefaultProvider("implied")
	terraformProvider := addrs.NewBuiltInProvider("terraform")
	configuredProvider := addrs.NewDefaultProvider("configured")
	grandchildProvider := addrs.NewDefaultProvider("grandchild")

	got, diags := cfg.ProviderRequirements()
	assertNoDiagnostics(t, diags)
	want := getproviders.Requirements{
		// the nullProvider constraints from the two modules are merged
		nullProvider:       getproviders.MustParseVersionConstraints("~> 2.0.0, 2.0.1"),
		randomProvider:     getproviders.MustParseVersionConstraints("~> 1.2.0"),
		tlsProvider:        getproviders.MustParseVersionConstraints("~> 3.0"),
		configuredProvider: getproviders.MustParseVersionConstraints("~> 1.4"),
		impliedProvider:    nil,
		happycloudProvider: nil,
		terraformProvider:  nil,
		grandchildProvider: nil,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestConfigProviderRequirementsInclTests(t *testing.T) {
	cfg, diags := testNestedModuleConfigFromDirWithTests(t, "testdata/provider-reqs-with-tests")
	// TODO: Version Constraint Deprecation.
	// Once we've removed the version argument from provider configuration
	// blocks, this can go back to expected 0 diagnostics.
	// assertNoDiagnostics(t, diags)
	assertDiagnosticCount(t, diags, 1)
	assertDiagnosticSummary(t, diags, "Version constraints inside provider configuration blocks are deprecated")

	tlsProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost,
		"hashicorp", "tls",
	)
	nullProvider := addrs.NewDefaultProvider("null")
	randomProvider := addrs.NewDefaultProvider("random")
	impliedProvider := addrs.NewDefaultProvider("implied")
	terraformProvider := addrs.NewBuiltInProvider("terraform")
	configuredProvider := addrs.NewDefaultProvider("configured")

	got, diags := cfg.ProviderRequirements()
	assertNoDiagnostics(t, diags)
	want := getproviders.Requirements{
		// the nullProvider constraints from the two modules are merged
		nullProvider:       getproviders.MustParseVersionConstraints("~> 2.0.0"),
		randomProvider:     getproviders.MustParseVersionConstraints("~> 1.2.0"),
		tlsProvider:        getproviders.MustParseVersionConstraints("~> 3.0"),
		configuredProvider: getproviders.MustParseVersionConstraints("~> 1.4"),
		impliedProvider:    nil,
		terraformProvider:  nil,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestConfigProviderRequirementsDuplicate(t *testing.T) {
	_, diags := testNestedModuleConfigFromDir(t, "testdata/duplicate-local-name")
	assertDiagnosticCount(t, diags, 3)
	assertDiagnosticSummary(t, diags, "Duplicate required provider")
}

func TestConfigProviderRequirementsShallow(t *testing.T) {
	cfg, diags := testNestedModuleConfigFromDir(t, "testdata/provider-reqs")
	// TODO: Version Constraint Deprecation.
	// Once we've removed the version argument from provider configuration
	// blocks, this can go back to expected 0 diagnostics.
	// assertNoDiagnostics(t, diags)
	assertDiagnosticCount(t, diags, 1)
	assertDiagnosticSummary(t, diags, "Version constraints inside provider configuration blocks are deprecated")

	tlsProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost,
		"hashicorp", "tls",
	)
	nullProvider := addrs.NewDefaultProvider("null")
	randomProvider := addrs.NewDefaultProvider("random")
	impliedProvider := addrs.NewDefaultProvider("implied")
	terraformProvider := addrs.NewBuiltInProvider("terraform")
	configuredProvider := addrs.NewDefaultProvider("configured")

	got, diags := cfg.ProviderRequirementsShallow()
	assertNoDiagnostics(t, diags)
	want := getproviders.Requirements{
		// the nullProvider constraint is only from the root module
		nullProvider:       getproviders.MustParseVersionConstraints("~> 2.0.0"),
		randomProvider:     getproviders.MustParseVersionConstraints("~> 1.2.0"),
		tlsProvider:        getproviders.MustParseVersionConstraints("~> 3.0"),
		configuredProvider: getproviders.MustParseVersionConstraints("~> 1.4"),
		impliedProvider:    nil,
		terraformProvider:  nil,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestConfigProviderRequirementsShallowInclTests(t *testing.T) {
	cfg, diags := testNestedModuleConfigFromDirWithTests(t, "testdata/provider-reqs-with-tests")
	// TODO: Version Constraint Deprecation.
	// Once we've removed the version argument from provider configuration
	// blocks, this can go back to expected 0 diagnostics.
	// assertNoDiagnostics(t, diags)
	assertDiagnosticCount(t, diags, 1)
	assertDiagnosticSummary(t, diags, "Version constraints inside provider configuration blocks are deprecated")

	tlsProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost,
		"hashicorp", "tls",
	)
	impliedProvider := addrs.NewDefaultProvider("implied")
	terraformProvider := addrs.NewBuiltInProvider("terraform")
	configuredProvider := addrs.NewDefaultProvider("configured")

	got, diags := cfg.ProviderRequirementsShallow()
	assertNoDiagnostics(t, diags)
	want := getproviders.Requirements{
		tlsProvider:        getproviders.MustParseVersionConstraints("~> 3.0"),
		configuredProvider: getproviders.MustParseVersionConstraints("~> 1.4"),
		impliedProvider:    nil,
		terraformProvider:  nil,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestConfigProviderRequirementsByModule(t *testing.T) {
	cfg, diags := testNestedModuleConfigFromDir(t, "testdata/provider-reqs")
	// TODO: Version Constraint Deprecation.
	// Once we've removed the version argument from provider configuration
	// blocks, this can go back to expected 0 diagnostics.
	// assertNoDiagnostics(t, diags)
	assertDiagnosticCount(t, diags, 1)
	assertDiagnosticSummary(t, diags, "Version constraints inside provider configuration blocks are deprecated")

	tlsProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost,
		"hashicorp", "tls",
	)
	happycloudProvider := addrs.NewProvider(
		svchost.Hostname("tf.example.com"),
		"awesomecorp", "happycloud",
	)
	nullProvider := addrs.NewDefaultProvider("null")
	randomProvider := addrs.NewDefaultProvider("random")
	impliedProvider := addrs.NewDefaultProvider("implied")
	terraformProvider := addrs.NewBuiltInProvider("terraform")
	configuredProvider := addrs.NewDefaultProvider("configured")
	grandchildProvider := addrs.NewDefaultProvider("grandchild")

	got, diags := cfg.ProviderRequirementsByModule()
	assertNoDiagnostics(t, diags)
	want := &ModuleRequirements{
		Name:       "",
		SourceAddr: nil,
		SourceDir:  "testdata/provider-reqs",
		Requirements: getproviders.Requirements{
			// Only the root module's version is present here
			nullProvider:       getproviders.MustParseVersionConstraints("~> 2.0.0"),
			randomProvider:     getproviders.MustParseVersionConstraints("~> 1.2.0"),
			tlsProvider:        getproviders.MustParseVersionConstraints("~> 3.0"),
			configuredProvider: getproviders.MustParseVersionConstraints("~> 1.4"),
			impliedProvider:    nil,
			terraformProvider:  nil,
		},
		Children: map[string]*ModuleRequirements{
			"kinder": {
				Name:       "kinder",
				SourceAddr: addrs.ModuleSourceLocal("./child"),
				SourceDir:  "testdata/provider-reqs/child",
				Requirements: getproviders.Requirements{
					nullProvider:       getproviders.MustParseVersionConstraints("= 2.0.1"),
					happycloudProvider: nil,
				},
				Children: map[string]*ModuleRequirements{
					"nested": {
						Name:       "nested",
						SourceAddr: addrs.ModuleSourceLocal("./grandchild"),
						SourceDir:  "testdata/provider-reqs/child/grandchild",
						Requirements: getproviders.Requirements{
							grandchildProvider: nil,
						},
						Children: map[string]*ModuleRequirements{},
						Tests:    make(map[string]*TestFileModuleRequirements),
					},
				},
				Tests: make(map[string]*TestFileModuleRequirements),
			},
		},
		Tests: make(map[string]*TestFileModuleRequirements),
	}

	ignore := cmpopts.IgnoreUnexported(version.Constraint{}, cty.Value{}, hclsyntax.Body{})
	if diff := cmp.Diff(want, got, ignore); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestConfigProviderRequirementsByModuleInclTests(t *testing.T) {
	cfg, diags := testNestedModuleConfigFromDirWithTests(t, "testdata/provider-reqs-with-tests")
	// TODO: Version Constraint Deprecation.
	// Once we've removed the version argument from provider configuration
	// blocks, this can go back to expected 0 diagnostics.
	// assertNoDiagnostics(t, diags)
	assertDiagnosticCount(t, diags, 1)
	assertDiagnosticSummary(t, diags, "Version constraints inside provider configuration blocks are deprecated")

	tlsProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost,
		"hashicorp", "tls",
	)
	nullProvider := addrs.NewDefaultProvider("null")
	randomProvider := addrs.NewDefaultProvider("random")
	impliedProvider := addrs.NewDefaultProvider("implied")
	terraformProvider := addrs.NewBuiltInProvider("terraform")
	configuredProvider := addrs.NewDefaultProvider("configured")

	got, diags := cfg.ProviderRequirementsByModule()
	assertNoDiagnostics(t, diags)
	want := &ModuleRequirements{
		Name:       "",
		SourceAddr: nil,
		SourceDir:  "testdata/provider-reqs-with-tests",
		Requirements: getproviders.Requirements{
			// Only the root module's version is present here
			tlsProvider:       getproviders.MustParseVersionConstraints("~> 3.0"),
			impliedProvider:   nil,
			terraformProvider: nil,
		},
		Children: make(map[string]*ModuleRequirements),
		Tests: map[string]*TestFileModuleRequirements{
			"provider-reqs-root.tftest.hcl": {
				Requirements: getproviders.Requirements{
					configuredProvider: getproviders.MustParseVersionConstraints("~> 1.4"),
				},
				Runs: map[string]*ModuleRequirements{
					"setup": {
						Name:       "setup",
						SourceAddr: addrs.ModuleSourceLocal("./setup"),
						SourceDir:  "testdata/provider-reqs-with-tests/setup",
						Requirements: getproviders.Requirements{
							nullProvider:   getproviders.MustParseVersionConstraints("~> 2.0.0"),
							randomProvider: getproviders.MustParseVersionConstraints("~> 1.2.0"),
						},
						Children: make(map[string]*ModuleRequirements),
						Tests:    make(map[string]*TestFileModuleRequirements),
					},
				},
			},
		},
	}

	ignore := cmpopts.IgnoreUnexported(version.Constraint{}, cty.Value{}, hclsyntax.Body{})
	if diff := cmp.Diff(want, got, ignore); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestVerifyDependencySelections(t *testing.T) {
	cfg, diags := testNestedModuleConfigFromDir(t, "testdata/provider-reqs")
	// TODO: Version Constraint Deprecation.
	// Once we've removed the version argument from provider configuration
	// blocks, this can go back to expected 0 diagnostics.
	// assertNoDiagnostics(t, diags)
	assertDiagnosticCount(t, diags, 1)
	assertDiagnosticSummary(t, diags, "Version constraints inside provider configuration blocks are deprecated")

	tlsProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost,
		"hashicorp", "tls",
	)
	happycloudProvider := addrs.NewProvider(
		svchost.Hostname("tf.example.com"),
		"awesomecorp", "happycloud",
	)
	nullProvider := addrs.NewDefaultProvider("null")
	randomProvider := addrs.NewDefaultProvider("random")
	impliedProvider := addrs.NewDefaultProvider("implied")
	configuredProvider := addrs.NewDefaultProvider("configured")
	grandchildProvider := addrs.NewDefaultProvider("grandchild")

	tests := map[string]struct {
		PrepareLocks func(*depsfile.Locks)
		WantErrs     []string
	}{
		"empty locks": {
			func(*depsfile.Locks) {
				// Intentionally blank
			},
			[]string{
				`provider registry.terraform.io/hashicorp/configured: required by this configuration but no version is selected`,
				`provider registry.terraform.io/hashicorp/grandchild: required by this configuration but no version is selected`,
				`provider registry.terraform.io/hashicorp/implied: required by this configuration but no version is selected`,
				`provider registry.terraform.io/hashicorp/null: required by this configuration but no version is selected`,
				`provider registry.terraform.io/hashicorp/random: required by this configuration but no version is selected`,
				`provider registry.terraform.io/hashicorp/tls: required by this configuration but no version is selected`,
				`provider tf.example.com/awesomecorp/happycloud: required by this configuration but no version is selected`,
			},
		},
		"suitable locks": {
			func(locks *depsfile.Locks) {
				locks.SetProvider(configuredProvider, getproviders.MustParseVersion("1.4.0"), nil, nil)
				locks.SetProvider(grandchildProvider, getproviders.MustParseVersion("0.1.0"), nil, nil)
				locks.SetProvider(impliedProvider, getproviders.MustParseVersion("0.2.0"), nil, nil)
				locks.SetProvider(nullProvider, getproviders.MustParseVersion("2.0.1"), nil, nil)
				locks.SetProvider(randomProvider, getproviders.MustParseVersion("1.2.2"), nil, nil)
				locks.SetProvider(tlsProvider, getproviders.MustParseVersion("3.0.1"), nil, nil)
				locks.SetProvider(happycloudProvider, getproviders.MustParseVersion("0.0.1"), nil, nil)
			},
			nil,
		},
		"null provider constraints changed": {
			func(locks *depsfile.Locks) {
				locks.SetProvider(configuredProvider, getproviders.MustParseVersion("1.4.0"), nil, nil)
				locks.SetProvider(grandchildProvider, getproviders.MustParseVersion("0.1.0"), nil, nil)
				locks.SetProvider(impliedProvider, getproviders.MustParseVersion("0.2.0"), nil, nil)
				locks.SetProvider(nullProvider, getproviders.MustParseVersion("3.0.0"), nil, nil)
				locks.SetProvider(randomProvider, getproviders.MustParseVersion("1.2.2"), nil, nil)
				locks.SetProvider(tlsProvider, getproviders.MustParseVersion("3.0.1"), nil, nil)
				locks.SetProvider(happycloudProvider, getproviders.MustParseVersion("0.0.1"), nil, nil)
			},
			[]string{
				`provider registry.terraform.io/hashicorp/null: locked version selection 3.0.0 doesn't match the updated version constraints "~> 2.0.0, 2.0.1"`,
			},
		},
		"null provider lock changed": {
			func(locks *depsfile.Locks) {
				// In this case, we set the lock file version constraints to
				// match the configuration, and so our error message changes
				// to not assume the configuration changed anymore.
				locks.SetProvider(nullProvider, getproviders.MustParseVersion("3.0.0"), getproviders.MustParseVersionConstraints("~> 2.0.0, 2.0.1"), nil)

				locks.SetProvider(configuredProvider, getproviders.MustParseVersion("1.4.0"), nil, nil)
				locks.SetProvider(grandchildProvider, getproviders.MustParseVersion("0.1.0"), nil, nil)
				locks.SetProvider(impliedProvider, getproviders.MustParseVersion("0.2.0"), nil, nil)
				locks.SetProvider(randomProvider, getproviders.MustParseVersion("1.2.2"), nil, nil)
				locks.SetProvider(tlsProvider, getproviders.MustParseVersion("3.0.1"), nil, nil)
				locks.SetProvider(happycloudProvider, getproviders.MustParseVersion("0.0.1"), nil, nil)
			},
			[]string{
				`provider registry.terraform.io/hashicorp/null: version constraints "~> 2.0.0, 2.0.1" don't match the locked version selection 3.0.0`,
			},
		},
		"overridden provider": {
			func(locks *depsfile.Locks) {
				locks.SetProviderOverridden(happycloudProvider)
			},
			[]string{
				// We still catch all of the other ones, because only happycloud was overridden
				`provider registry.terraform.io/hashicorp/configured: required by this configuration but no version is selected`,
				`provider registry.terraform.io/hashicorp/grandchild: required by this configuration but no version is selected`,
				`provider registry.terraform.io/hashicorp/implied: required by this configuration but no version is selected`,
				`provider registry.terraform.io/hashicorp/null: required by this configuration but no version is selected`,
				`provider registry.terraform.io/hashicorp/random: required by this configuration but no version is selected`,
				`provider registry.terraform.io/hashicorp/tls: required by this configuration but no version is selected`,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			depLocks := depsfile.NewLocks()
			test.PrepareLocks(depLocks)
			gotErrs := cfg.VerifyDependencySelections(depLocks)

			var gotErrsStr []string
			if gotErrs != nil {
				gotErrsStr = make([]string, len(gotErrs))
				for i, err := range gotErrs {
					gotErrsStr[i] = err.Error()
				}
			}

			if diff := cmp.Diff(test.WantErrs, gotErrsStr); diff != "" {
				t.Errorf("wrong errors\n%s", diff)
			}
		})
	}
}

func TestConfigProviderForConfigAddr(t *testing.T) {
	cfg, diags := testModuleConfigFromDir("testdata/valid-modules/providers-fqns")
	assertNoDiagnostics(t, diags)

	got := cfg.ProviderForConfigAddr(addrs.NewDefaultLocalProviderConfig("foo-test"))
	want := addrs.NewProvider(addrs.DefaultProviderRegistryHost, "foo", "test")
	if !got.Equals(want) {
		t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
	}

	// now check a provider that isn't in the configuration. It should return a DefaultProvider.
	got = cfg.ProviderForConfigAddr(addrs.NewDefaultLocalProviderConfig("bar-test"))
	want = addrs.NewDefaultProvider("bar-test")
	if !got.Equals(want) {
		t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
	}
}

func TestConfigAddProviderRequirements(t *testing.T) {
	cfg, diags := testModuleConfigFromFile("testdata/valid-files/providers-explicit-implied.tf")
	assertNoDiagnostics(t, diags)

	reqs := getproviders.Requirements{
		addrs.NewDefaultProvider("null"): nil,
	}
	diags = cfg.addProviderRequirements(reqs, true, false)
	assertNoDiagnostics(t, diags)
}

func TestConfigImportProviderClashesWithModules(t *testing.T) {
	src, err := os.ReadFile("testdata/invalid-import-files/import-and-module-clash.tf")
	if err != nil {
		t.Fatal(err)
	}

	parser := testParser(map[string]string{
		"main.tf": string(src),
	})

	_, diags := parser.LoadConfigFile("main.tf")
	assertExactDiagnostics(t, diags, []string{
		`main.tf:9,3-19: Invalid import provider argument; The provider argument can only be specified in import blocks that will generate configuration.

Use the providers argument within the module block to configure providers for all resources within a module, including imported resources.`,
	})
}

func TestConfigImportProviderClashesWithResources(t *testing.T) {
	cfg, diags := testModuleConfigFromFile("testdata/invalid-import-files/import-and-resource-clash.tf")
	assertNoDiagnostics(t, diags)

	diags = cfg.addProviderRequirements(getproviders.Requirements{}, true, false)
	assertExactDiagnostics(t, diags, []string{
		`testdata/invalid-import-files/import-and-resource-clash.tf:9,3-19: Invalid import provider argument; The provider argument can only be specified in import blocks that will generate configuration.

Use the provider argument in the target resource block to configure the provider for a resource with explicit provider configuration.`,
	})
}

func TestConfigImportProviderWithNoResourceProvider(t *testing.T) {
	cfg, diags := testModuleConfigFromFile("testdata/invalid-import-files/import-and-no-resource.tf")
	assertNoDiagnostics(t, diags)

	diags = cfg.addProviderRequirements(getproviders.Requirements{}, true, false)
	assertExactDiagnostics(t, diags, []string{
		`testdata/invalid-import-files/import-and-no-resource.tf:5,3-19: Invalid import provider argument; The provider argument can only be specified in import blocks that will generate configuration.

Use the provider argument in the target resource block to configure the provider for a resource with explicit provider configuration.`,
	})
}

func TestTransformForTest(t *testing.T) {

	str := func(providers map[string]string) string {
		var buffer bytes.Buffer
		for key, config := range providers {
			buffer.WriteString(fmt.Sprintf("%s: %s\n", key, config))
		}
		return buffer.String()
	}

	convertToProviders := func(t *testing.T, contents map[string]string) map[string]*Provider {
		t.Helper()

		providers := make(map[string]*Provider)
		for key, content := range contents {
			parser := hclparse.NewParser()
			file, diags := parser.ParseHCL([]byte(content), fmt.Sprintf("%s.hcl", key))
			if diags.HasErrors() {
				t.Fatal(diags.Error())
			}

			provider := &Provider{
				Config: file.Body,
			}

			parts := strings.Split(key, ".")
			provider.Name = parts[0]
			if len(parts) > 1 {
				provider.Alias = parts[1]
			}

			providers[key] = provider
		}
		return providers
	}

	validate := func(t *testing.T, msg string, expected map[string]string, actual map[string]*Provider) {
		t.Helper()

		converted := make(map[string]string)
		for key, provider := range actual {
			content, err := provider.Config.Content(&hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{Name: "source", Required: true},
				},
			})
			if err != nil {
				t.Fatal(err)
			}

			source, diags := content.Attributes["source"].Expr.Value(nil)
			if diags.HasErrors() {
				t.Fatal(diags.Error())
			}
			converted[key] = fmt.Sprintf("source = %q", source.AsString())
		}

		if diff := cmp.Diff(expected, converted); len(diff) > 0 {
			t.Errorf("%s\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", msg, str(expected), str(converted), diff)
		}
	}

	tcs := map[string]struct {
		configProviders   map[string]string
		fileProviders     map[string]string
		runProviders      []PassedProviderConfig
		expectedProviders map[string]string
		expectedErrors    []string
	}{
		"empty": {
			configProviders:   make(map[string]string),
			expectedProviders: make(map[string]string),
		},
		"only providers in config": {
			configProviders: map[string]string{
				"foo": "source = \"config\"",
				"bar": "source = \"config\"",
			},
			expectedProviders: map[string]string{
				"foo": "source = \"config\"",
				"bar": "source = \"config\"",
			},
		},
		"only providers in test file": {
			configProviders: make(map[string]string),
			fileProviders: map[string]string{
				"foo": "source = \"testfile\"",
				"bar": "source = \"testfile\"",
			},
			expectedProviders: map[string]string{
				"foo": "source = \"testfile\"",
				"bar": "source = \"testfile\"",
			},
		},
		"only providers in run block": {
			configProviders: make(map[string]string),
			runProviders: []PassedProviderConfig{
				{
					InChild: &ProviderConfigRef{
						Name: "foo",
					},
					InParent: &ProviderConfigRef{
						Name: "bar",
					},
				},
			},
			expectedProviders: make(map[string]string),
			expectedErrors: []string{
				":0,0-0: Missing provider definition for bar; This provider block references a provider definition that does not exist.",
			},
		},
		"subset of providers in test file": {
			configProviders: make(map[string]string),
			fileProviders: map[string]string{
				"bar": "source = \"testfile\"",
			},
			runProviders: []PassedProviderConfig{
				{
					InChild: &ProviderConfigRef{
						Name: "foo",
					},
					InParent: &ProviderConfigRef{
						Name: "bar",
					},
				},
			},
			expectedProviders: map[string]string{
				"foo": "source = \"testfile\"",
			},
		},
		"overrides providers in config": {
			configProviders: map[string]string{
				"foo": "source = \"config\"",
				"bar": "source = \"config\"",
			},
			fileProviders: map[string]string{
				"bar": "source = \"testfile\"",
			},
			expectedProviders: map[string]string{
				"foo": "source = \"config\"",
				"bar": "source = \"testfile\"",
			},
		},
		"overrides subset of providers in config": {
			configProviders: map[string]string{
				"foo": "source = \"config\"",
				"bar": "source = \"config\"",
			},
			fileProviders: map[string]string{
				"foo": "source = \"testfile\"",
				"bar": "source = \"testfile\"",
			},
			runProviders: []PassedProviderConfig{
				{
					InChild: &ProviderConfigRef{
						Name: "bar",
					},
					InParent: &ProviderConfigRef{
						Name: "bar",
					},
				},
			},
			expectedProviders: map[string]string{
				"foo": "source = \"config\"",
				"bar": "source = \"testfile\"",
			},
		},
		"handles aliases": {
			configProviders: map[string]string{
				"foo.primary":   "source = \"config\"",
				"foo.secondary": "source = \"config\"",
			},
			fileProviders: map[string]string{
				"foo": "source = \"testfile\"",
			},
			runProviders: []PassedProviderConfig{
				{
					InChild: &ProviderConfigRef{
						Name: "foo.secondary",
					},
					InParent: &ProviderConfigRef{
						Name: "foo",
					},
				},
			},
			expectedProviders: map[string]string{
				"foo.primary":   "source = \"config\"",
				"foo.secondary": "source = \"testfile\"",
			},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			config := &Config{
				Module: &Module{
					ProviderConfigs: convertToProviders(t, tc.configProviders),
				},
			}

			file := &TestFile{
				Providers: convertToProviders(t, tc.fileProviders),
			}

			run := &TestRun{
				Providers: tc.runProviders,
			}

			reset, diags := config.TransformForTest(run, file)

			var actualErrs []string
			for _, err := range diags.Errs() {
				actualErrs = append(actualErrs, err.Error())
			}
			if diff := cmp.Diff(actualErrs, tc.expectedErrors, cmpopts.IgnoreUnexported()); len(diff) > 0 {
				t.Errorf("unmatched errors\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", strings.Join(tc.expectedErrors, "\n"), strings.Join(actualErrs, "\n"), diff)
			}

			validate(t, "after transform mismatch", tc.expectedProviders, config.Module.ProviderConfigs)
			reset()
			validate(t, "after reset mismatch", tc.configProviders, config.Module.ProviderConfigs)

		})
	}
}

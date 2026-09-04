// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/go-test/deep"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/spf13/afero"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	_ "github.com/hashicorp/terraform/internal/logging"
)

func TestConfigProviderTypes(t *testing.T) {
	// nil cfg should return an empty map
	got := configs.NewEmptyConfig().ProviderTypes()
	if len(got) != 0 {
		t.Fatal("expected empty result from empty config")
	}

	cfg, diags := testModuleCfgFromFileWithExperiments("testdata/config-graph/valid-files/providers-explicit-implied.tf")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	got = cfg.ProviderTypes()
	want := []addrs.Provider{
		addrs.NewDefaultProvider("aws"),
		addrs.NewDefaultProvider("external"),
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
	c := configs.NewEmptyConfig()
	got := c.ProviderTypes()
	if len(got) != 0 {
		t.Fatalf("wrong result!\ngot: %#v\nwant: nil\n", got)
	}

	// config with two provider sources, and one implicit (default) provider
	cfg, diags := testNestedModuleConfigFromDir(t, "testdata/config-graph/valid-modules/nested-providers-fqns")
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
	cfg, diags := testModuleConfigFromDir("testdata/config-graph/providers-explicit-fqn")
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
	cfg, diags := testNestedModuleConfigFromDir(t, "testdata/config-graph/provider-reqs")
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
	want := providerreqs.Requirements{
		// the nullProvider constraints from the two modules are merged
		nullProvider:       providerreqs.MustParseVersionConstraints("~> 2.0.0, 2.0.1"),
		randomProvider:     providerreqs.MustParseVersionConstraints("~> 1.2.0"),
		tlsProvider:        providerreqs.MustParseVersionConstraints("~> 3.0"),
		configuredProvider: providerreqs.MustParseVersionConstraints("~> 1.4"),
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
	cfg, diags := testNestedModuleConfigFromDirWithTests(t, "testdata/config-graph/provider-reqs-with-tests")
	assertDiagnosticCount(t, diags, 0)

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
	want := providerreqs.Requirements{
		// the nullProvider constraints from the two modules are merged
		nullProvider:       providerreqs.MustParseVersionConstraints("~> 2.0.0"),
		randomProvider:     providerreqs.MustParseVersionConstraints("~> 1.2.0"),
		tlsProvider:        providerreqs.MustParseVersionConstraints("~> 3.0"),
		configuredProvider: nil,
		impliedProvider:    nil,
		terraformProvider:  nil,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestConfigProviderRequirementsDuplicate(t *testing.T) {
	_, diags := testNestedModuleConfigFromDir(t, "testdata/config-graph/duplicate-local-name")
	assertDiagnosticCount(t, diags, 3)
	assertDiagnosticSummary(t, diags, "Duplicate required provider")
}

func TestConfigProviderRequirementsShallow(t *testing.T) {
	cfg, diags := testNestedModuleConfigFromDir(t, "testdata/config-graph/provider-reqs")
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
	want := providerreqs.Requirements{
		// the nullProvider constraint is only from the root module
		nullProvider:       providerreqs.MustParseVersionConstraints("~> 2.0.0"),
		randomProvider:     providerreqs.MustParseVersionConstraints("~> 1.2.0"),
		tlsProvider:        providerreqs.MustParseVersionConstraints("~> 3.0"),
		configuredProvider: providerreqs.MustParseVersionConstraints("~> 1.4"),
		impliedProvider:    nil,
		terraformProvider:  nil,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestConfigProviderRequirementsShallowInclTests(t *testing.T) {
	cfg, diags := testNestedModuleConfigFromDirWithTests(t, "testdata/config-graph/provider-reqs-with-tests")
	assertDiagnosticCount(t, diags, 0)

	tlsProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost,
		"hashicorp", "tls",
	)
	impliedProvider := addrs.NewDefaultProvider("implied")
	terraformProvider := addrs.NewBuiltInProvider("terraform")

	got, diags := cfg.ProviderRequirementsShallow()
	assertNoDiagnostics(t, diags)
	want := providerreqs.Requirements{
		tlsProvider:       providerreqs.MustParseVersionConstraints("~> 3.0"),
		impliedProvider:   nil,
		terraformProvider: nil,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestConfigProviderRequirementsByModule(t *testing.T) {
	cfg, diags := testNestedModuleConfigFromDir(t, "testdata/config-graph/provider-reqs")
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
	want := &configs.ModuleRequirements{
		Name:       "",
		SourceAddr: nil,
		SourceDir:  "testdata/config-graph/provider-reqs",
		Requirements: providerreqs.Requirements{
			// Only the root module's version is present here
			nullProvider:       providerreqs.MustParseVersionConstraints("~> 2.0.0"),
			randomProvider:     providerreqs.MustParseVersionConstraints("~> 1.2.0"),
			tlsProvider:        providerreqs.MustParseVersionConstraints("~> 3.0"),
			configuredProvider: providerreqs.MustParseVersionConstraints("~> 1.4"),
			impliedProvider:    nil,
			terraformProvider:  nil,
		},
		Children: map[string]*configs.ModuleRequirements{
			"kinder": {
				Name:       "kinder",
				SourceAddr: addrs.ModuleSourceLocal("./child"),
				SourceDir:  "testdata/config-graph/provider-reqs/child",
				Requirements: providerreqs.Requirements{
					nullProvider:       providerreqs.MustParseVersionConstraints("= 2.0.1"),
					happycloudProvider: nil,
				},
				Children: map[string]*configs.ModuleRequirements{
					"nested": {
						Name:       "nested",
						SourceAddr: addrs.ModuleSourceLocal("./grandchild"),
						SourceDir:  "testdata/config-graph/provider-reqs/child/grandchild",
						Requirements: providerreqs.Requirements{
							grandchildProvider: nil,
						},
						Children: map[string]*configs.ModuleRequirements{},
						Tests:    make(map[string]*configs.TestFileModuleRequirements),
					},
				},
				Tests: make(map[string]*configs.TestFileModuleRequirements),
			},
		},
		Tests: make(map[string]*configs.TestFileModuleRequirements),
	}

	ignore := cmpopts.IgnoreUnexported(version.Constraint{}, cty.Value{}, hclsyntax.Body{})
	if diff := cmp.Diff(want, got, ignore); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestConfigProviderRequirementsByModuleInclTests(t *testing.T) {
	cfg, diags := testNestedModuleConfigFromDirWithTests(t, "testdata/config-graph/provider-reqs-with-tests")
	assertDiagnosticCount(t, diags, 0)

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
	want := &configs.ModuleRequirements{
		Name:       "",
		SourceAddr: nil,
		SourceDir:  "testdata/config-graph/provider-reqs-with-tests",
		Requirements: providerreqs.Requirements{
			// Only the root module's version is present here
			tlsProvider:       providerreqs.MustParseVersionConstraints("~> 3.0"),
			impliedProvider:   nil,
			terraformProvider: nil,
		},
		Children: make(map[string]*configs.ModuleRequirements),
		Tests: map[string]*configs.TestFileModuleRequirements{
			"provider-reqs-root.tftest.hcl": {
				Requirements: providerreqs.Requirements{},
				Runs: map[string]*configs.ModuleRequirements{
					"setup": {
						Name:       "setup",
						SourceAddr: addrs.ModuleSourceLocal("./setup"),
						SourceDir:  "testdata/config-graph/provider-reqs-with-tests/setup",
						Requirements: providerreqs.Requirements{
							nullProvider:       providerreqs.MustParseVersionConstraints("~> 2.0.0"),
							randomProvider:     providerreqs.MustParseVersionConstraints("~> 1.2.0"),
							configuredProvider: nil,
						},
						Children: make(map[string]*configs.ModuleRequirements),
						Tests:    make(map[string]*configs.TestFileModuleRequirements),
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
	cfg, diags := testNestedModuleConfigFromDir(t, "testdata/config-graph/provider-reqs")
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
				locks.SetProvider(configuredProvider, providerreqs.MustParseVersion("1.4.0"), nil, nil)
				locks.SetProvider(grandchildProvider, providerreqs.MustParseVersion("0.1.0"), nil, nil)
				locks.SetProvider(impliedProvider, providerreqs.MustParseVersion("0.2.0"), nil, nil)
				locks.SetProvider(nullProvider, providerreqs.MustParseVersion("2.0.1"), nil, nil)
				locks.SetProvider(randomProvider, providerreqs.MustParseVersion("1.2.2"), nil, nil)
				locks.SetProvider(tlsProvider, providerreqs.MustParseVersion("3.0.1"), nil, nil)
				locks.SetProvider(happycloudProvider, providerreqs.MustParseVersion("0.0.1"), nil, nil)
			},
			nil,
		},
		"null provider constraints changed": {
			func(locks *depsfile.Locks) {
				locks.SetProvider(configuredProvider, providerreqs.MustParseVersion("1.4.0"), nil, nil)
				locks.SetProvider(grandchildProvider, providerreqs.MustParseVersion("0.1.0"), nil, nil)
				locks.SetProvider(impliedProvider, providerreqs.MustParseVersion("0.2.0"), nil, nil)
				locks.SetProvider(nullProvider, providerreqs.MustParseVersion("3.0.0"), nil, nil)
				locks.SetProvider(randomProvider, providerreqs.MustParseVersion("1.2.2"), nil, nil)
				locks.SetProvider(tlsProvider, providerreqs.MustParseVersion("3.0.1"), nil, nil)
				locks.SetProvider(happycloudProvider, providerreqs.MustParseVersion("0.0.1"), nil, nil)
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
				locks.SetProvider(nullProvider, providerreqs.MustParseVersion("3.0.0"), providerreqs.MustParseVersionConstraints("~> 2.0.0, 2.0.1"), nil)

				locks.SetProvider(configuredProvider, providerreqs.MustParseVersion("1.4.0"), nil, nil)
				locks.SetProvider(grandchildProvider, providerreqs.MustParseVersion("0.1.0"), nil, nil)
				locks.SetProvider(impliedProvider, providerreqs.MustParseVersion("0.2.0"), nil, nil)
				locks.SetProvider(randomProvider, providerreqs.MustParseVersion("1.2.2"), nil, nil)
				locks.SetProvider(tlsProvider, providerreqs.MustParseVersion("3.0.1"), nil, nil)
				locks.SetProvider(happycloudProvider, providerreqs.MustParseVersion("0.0.1"), nil, nil)
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
	cfg, diags := testModuleConfigFromDir("testdata/config-graph/valid-modules/providers-fqns")
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
	cfg, diags := testModuleConfigFromFile("testdata/config-graph/valid-files/providers-explicit-implied.tf")
	assertNoDiagnostics(t, diags)

	reqs := providerreqs.Requirements{
		addrs.NewDefaultProvider("null"): nil,
	}
	diags = cfg.AddProviderRequirements(reqs, true, false)
	assertNoDiagnostics(t, diags)
}

func TestConfigImportProviderClashesWithModules(t *testing.T) {
	src, err := os.ReadFile("testdata/config-graph/invalid-import-files/import-and-module-clash.tf")
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
	cfg, diags := testModuleConfigFromFile("testdata/config-graph/invalid-import-files/import-and-resource-clash.tf")
	assertNoDiagnostics(t, diags)

	diags = cfg.AddProviderRequirements(providerreqs.Requirements{}, true, false)
	assertExactDiagnostics(t, diags, []string{
		`testdata/config-graph/invalid-import-files/import-and-resource-clash.tf:9,3-19: Invalid import provider argument; The provider argument can only be specified in import blocks that will generate configuration.

Use the provider argument in the target resource block to configure the provider for a resource with explicit provider configuration.`,
	})
}

func TestConfigImportProviderWithNoResourceProvider(t *testing.T) {
	cfg, diags := testModuleConfigFromFile("testdata/config-graph/invalid-import-files/import-and-no-resource.tf")
	assertNoDiagnostics(t, diags)

	diags = cfg.AddProviderRequirements(providerreqs.Requirements{}, true, false)
	assertExactDiagnostics(t, diags, []string{
		`testdata/config-graph/invalid-import-files/import-and-no-resource.tf:5,3-19: Invalid import provider argument; The provider argument can only be specified in import blocks that will generate configuration.

Use the provider argument in the target resource block to configure the provider for a resource with explicit provider configuration.`,
	})
}

func TestConfigActionInResourceDependsOn(t *testing.T) {
	src, err := os.ReadFile("testdata/config-graph/invalid-modules/action-in-depends_on/action-in-resource-depends_on.tf")
	if err != nil {
		t.Fatal(err)
	}

	parser := testParser(map[string]string{
		"main.tf": string(src),
	})

	_, diags := parser.LoadConfigFile("main.tf")
	assertExactDiagnostics(t, diags, []string{
		`main.tf:5,17-42: Invalid depends_on Action Reference; The depends_on attribute cannot reference action blocks directly. You must reference a resource or data source instead.`,
	})
}

// testNestedModuleConfigFromDirWithTests matches testNestedModuleConfigFromDir
// except it also loads any test files within the directory.
func testNestedModuleConfigFromDirWithTests(t *testing.T, path string) (*configs.Config, hcl.Diagnostics) {
	t.Helper()

	parser := configs.NewParser(nil)
	mod, diags := parser.LoadConfigDir(path, configs.MatchTestFiles("tests"))
	if mod == nil {
		t.Fatal("got nil root module; want non-nil")
	}

	cfg, nestedDiags := buildNestedModuleConfig(mod, path, parser)

	diags = append(diags, nestedDiags...)
	return cfg, diags
}

func buildNestedModuleConfig(mod *configs.Module, path string, parser *configs.Parser) (*configs.Config, hcl.Diagnostics) {
	versionI := 0

	walkerFunc := configs.ModuleWalkerFunc(
		func(req *configs.ModuleRequest) (*configs.Module, *version.Version, hcl.Diagnostics) {
			// For the sake of this test we're going to just treat our
			// SourceAddr as a path relative to the calling module.
			// A "real" implementation of ModuleWalker should accept the
			// various different source address syntaxes Terraform supports.

			// Build a full path by walking up the module tree, prepending each
			// source address path until we hit the root
			paths := []string{req.SourceAddr.String()}
			for config := req.Parent; config != nil && config.Parent != nil; config = config.Parent {
				paths = append([]string{config.SourceAddr.String()}, paths...)
			}
			paths = append([]string{path}, paths...)
			sourcePath := filepath.Join(paths...)

			mod, diags := parser.LoadConfigDir(sourcePath)
			version, _ := version.NewVersion(fmt.Sprintf("1.0.%d", versionI))
			versionI++
			return mod, version, diags
		})
	mockLoaderFunc := configs.MockDataLoaderFunc(func(provider *configs.Provider) (*configs.MockData, hcl.Diagnostics) {
		return nil, nil
	})

	cfg, diags := BuildConfigWithGraph(
		mod,
		walkerFunc,
		nil,
		mockLoaderFunc,
	)

	return cfg, diags.ToHCL()
}

// testModuleFromDir reads configuration from the given directory path as a
// module and returns its configuration. This is a helper for use in unit tests.
func testModuleConfigFromDir(path string) (*configs.Config, hcl.Diagnostics) {
	parser := configs.NewParser(nil)
	mod, diags := parser.LoadConfigDir(path)
	cfg := testConfig(mod)
	moreDiags := configs.FinalizeConfig(cfg, nil)
	return cfg, append(diags, moreDiags...)
}

func assertDiagnosticSummary(t *testing.T, diags hcl.Diagnostics, want string) bool {
	t.Helper()

	for _, diag := range diags {
		if diag.Summary == want {
			return false
		}
	}

	t.Errorf("missing diagnostic summary %q", want)
	for _, diag := range diags {
		t.Logf("- %s", diag)
	}
	return true
}

func testConfig(mod *configs.Module) *configs.Config {
	cfg := &configs.Config{Module: mod, Children: map[string]*configs.Config{}}
	cfg.Root = cfg
	return cfg
}

// testModuleConfigFrom File reads a single file from the given path as a
// module and returns its configuration. This is a helper for use in unit tests.
func testModuleConfigFromFile(filename string) (*configs.Config, hcl.Diagnostics) {
	parser := configs.NewParser(nil)
	f, diags := parser.LoadConfigFile(filename)
	mod, modDiags := configs.NewModule([]*configs.File{f}, nil)
	diags = append(diags, modDiags...)
	cfg := testConfig(mod)
	moreDiags := configs.FinalizeConfig(cfg, nil)
	return cfg, append(diags, moreDiags...)
}

// testModuleCfgFromFileWithExperiments File reads a single file from the given path as a
// module and returns its configuration. This is a helper for use in unit tests.
func testModuleCfgFromFileWithExperiments(filename string) (*configs.Config, hcl.Diagnostics) {
	parser := configs.NewParser(nil)
	parser.AllowLanguageExperiments(true)
	f, diags := parser.LoadConfigFile(filename)
	mod, modDiags := configs.NewModule([]*configs.File{f}, nil)
	diags = append(diags, modDiags...)
	cfg := testConfig(mod)
	moreDiags := configs.FinalizeConfig(cfg, nil)
	return cfg, append(diags, moreDiags...)
}

func assertExactDiagnostics(t *testing.T, diags hcl.Diagnostics, want []string) bool {
	t.Helper()

	gotDiags := map[string]bool{}
	wantDiags := map[string]bool{}

	for _, diag := range diags {
		gotDiags[diag.Error()] = true
	}
	for _, msg := range want {
		wantDiags[msg] = true
	}

	bad := false
	for got := range gotDiags {
		if _, exists := wantDiags[got]; !exists {
			t.Errorf("unexpected diagnostic: %s", got)
			bad = true
		}
	}
	for want := range wantDiags {
		if _, exists := gotDiags[want]; !exists {
			t.Errorf("missing expected diagnostic: %s", want)
			bad = true
		}
	}

	return bad
}

// testParser returns a parser that reads files from the given map, which
// is from paths to file contents.
//
// Since this function uses only in-memory objects, it should never fail.
// If any errors are encountered in practice, this function will panic.
func testParser(files map[string]string) *configs.Parser {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	for filePath, contents := range files {
		dirPath := path.Dir(filePath)
		err := fs.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			panic(err)
		}
		err = fs.WriteFile(filePath, []byte(contents), os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	return configs.NewParser(fs)
}

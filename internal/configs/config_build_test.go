// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"path"
	"path/filepath"
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
)

func TestBuildConfig_WithMockDataSources(t *testing.T) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDirWithTests("testdata/valid-modules/with-mock-sources", "tests")
	assertNoDiagnostics(t, diags)
	assertNoDiagnostics(t, diags)
	if mod == nil {
		t.Fatal("got nil root module; want non-nil")
	}

	cfg, diags := BuildConfig(mod, nil, MockDataLoaderFunc(func(provider *Provider) (*MockData, hcl.Diagnostics) {
		sourcePath := filepath.Join("testdata/valid-modules/with-mock-sources", provider.MockDataExternalSource)
		return parser.LoadMockDataDir(sourcePath, provider.MockDataDuringPlan, hcl.Range{})
	}))
	assertNoDiagnostics(t, diags)
	if cfg == nil {
		t.Fatal("got nil config; want non-nil")
	}

	provider := cfg.Module.Tests["main.tftest.hcl"].Providers["aws"]

	if len(provider.MockData.MockDataSources) != 1 {
		t.Errorf("expected to load 1 mock data source but loaded %d", len(provider.MockData.MockDataSources))
	}
	if len(provider.MockData.MockResources) != 1 {
		t.Errorf("expected to load 1 mock resource but loaded %d", len(provider.MockData.MockResources))
	}
	if provider.MockData.Overrides.Len() != 1 {
		t.Errorf("expected to load 1 override but loaded %d", provider.MockData.Overrides.Len())
	}
}

func TestBuildConfig_WithMockDataSourcesInline(t *testing.T) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDirWithTests("testdata/valid-modules/with-mock-sources-inline", "tests")
	assertNoDiagnostics(t, diags)
	assertNoDiagnostics(t, diags)
	if mod == nil {
		t.Fatal("got nil root module; want non-nil")
	}

	cfg, diags := BuildConfig(mod, nil, MockDataLoaderFunc(func(provider *Provider) (*MockData, hcl.Diagnostics) {
		sourcePath := filepath.Join("testdata/valid-modules/with-mock-sources-inline", provider.MockDataExternalSource)
		return parser.LoadMockDataDir(sourcePath, provider.MockDataDuringPlan, hcl.Range{})
	}))
	assertNoDiagnostics(t, diags)
	if cfg == nil {
		t.Fatal("got nil config; want non-nil")
	}
}

func TestBuildConfig_WithNestedTestModules(t *testing.T) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDirWithTests("testdata/valid-modules/with-tests-nested-module", "tests")
	assertNoDiagnostics(t, diags)
	if mod == nil {
		t.Fatal("got nil root module; want non-nil")
	}

	cfg, diags := BuildConfig(mod, ModuleWalkerFunc(
		func(req *ModuleRequest) (*Module, *version.Version, hcl.Diagnostics) {

			// Bit of a hack to get the test working, but we know all the source
			// addresses in this test are locals, so we can just treat them as
			// paths in the filesystem.

			addr := req.SourceAddr.String()
			current := req.Parent
			for current.SourceAddr != nil {
				addr = path.Join(current.SourceAddr.String(), addr)
				current = current.Parent
			}
			sourcePath := filepath.Join("testdata/valid-modules/with-tests-nested-module", addr)

			mod, diags := parser.LoadConfigDir(sourcePath)
			version, _ := version.NewVersion("1.0.0")
			return mod, version, diags
		}),
		MockDataLoaderFunc(func(provider *Provider) (*MockData, hcl.Diagnostics) {
			return nil, nil
		}),
	)
	assertNoDiagnostics(t, diags)
	if cfg == nil {
		t.Fatal("got nil config; want non-nil")
	}

	// We should have loaded our test case, and one of the test runs should
	// have loaded an alternate module.

	if len(cfg.Module.Tests) != 1 {
		t.Fatalf("expected exactly one test case but found %d", len(cfg.Module.Tests))
	}

	test := cfg.Module.Tests["main.tftest.hcl"]
	if len(test.Runs) != 1 {
		t.Fatalf("expected two test runs but found %d", len(test.Runs))
	}

	run := test.Runs[0]
	if run.ConfigUnderTest == nil {
		t.Fatalf("the first test run should have loaded config but did not")
	}

	if run.ConfigUnderTest.Parent != nil {
		t.Errorf("config under test should not have a parent")
	}

	if run.ConfigUnderTest.Root != run.ConfigUnderTest {
		t.Errorf("config under test root should be itself")
	}

	if len(run.ConfigUnderTest.Path) > 0 {
		t.Errorf("config under test path should be the root module")
	}

	// We should also have loaded a single child underneath the config under
	// test, and it should have valid paths.

	child := run.ConfigUnderTest.Children["child"]

	if child.Parent != run.ConfigUnderTest {
		t.Errorf("child should point back to root")
	}

	if len(child.Path) != 1 || child.Path[0] != "child" {
		t.Errorf("child should have rebased against virtual root")
	}

	if child.Root != run.ConfigUnderTest {
		t.Errorf("child root should be main config under test")
	}
}

func TestBuildConfig_WithTestModule(t *testing.T) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDirWithTests("testdata/valid-modules/with-tests-module", "tests")
	assertNoDiagnostics(t, diags)
	if mod == nil {
		t.Fatal("got nil root module; want non-nil")
	}

	cfg, diags := BuildConfig(mod, ModuleWalkerFunc(
		func(req *ModuleRequest) (*Module, *version.Version, hcl.Diagnostics) {
			// For the sake of this test we're going to just treat our
			// SourceAddr as a path relative to our fixture directory.
			// A "real" implementation of ModuleWalker should accept the
			// various different source address syntaxes Terraform supports.
			sourcePath := filepath.Join("testdata/valid-modules/with-tests-module", req.SourceAddr.String())

			mod, diags := parser.LoadConfigDir(sourcePath)
			version, _ := version.NewVersion("1.0.0")
			return mod, version, diags
		}),
		MockDataLoaderFunc(func(provider *Provider) (*MockData, hcl.Diagnostics) {
			return nil, nil
		}),
	)
	assertNoDiagnostics(t, diags)
	if cfg == nil {
		t.Fatal("got nil config; want non-nil")
	}

	// We should have loaded our test case, and one of the test runs should
	// have loaded an alternate module.

	if len(cfg.Module.Tests) != 1 {
		t.Fatalf("expected exactly one test case but found %d", len(cfg.Module.Tests))
	}

	test := cfg.Module.Tests["main.tftest.hcl"]
	if len(test.Runs) != 2 {
		t.Fatalf("expected two test runs but found %d", len(test.Runs))
	}

	run := test.Runs[0]
	if run.ConfigUnderTest == nil {
		t.Fatalf("the first test run should have loaded config but did not")
	}

	if run.ConfigUnderTest.Parent != nil {
		t.Errorf("config under test should not have a parent")
	}

	if run.ConfigUnderTest.Root != run.ConfigUnderTest {
		t.Errorf("config under test root should be itself")
	}

	if len(run.ConfigUnderTest.Path) > 0 {
		t.Errorf("config under test path should be the root module")
	}
}

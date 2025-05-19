// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/zclconf/go-cty/cty"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
)

func TestBuildConfig(t *testing.T) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDir("testdata/config-build")
	assertNoDiagnostics(t, diags)
	if mod == nil {
		t.Fatal("got nil root module; want non-nil")
	}

	versionI := 0
	cfg, diags := BuildConfig(mod, ModuleWalkerFunc(
		func(req *ModuleRequest) (*Module, *version.Version, hcl.Diagnostics) {
			// For the sake of this test we're going to just treat our
			// SourceAddr as a path relative to our fixture directory.
			// A "real" implementation of ModuleWalker should accept the
			// various different source address syntaxes Terraform supports.
			sourcePath := filepath.Join("testdata/config-build", req.SourceAddr.String())

			mod, diags := parser.LoadConfigDir(sourcePath)
			version, _ := version.NewVersion(fmt.Sprintf("1.0.%d", versionI))
			versionI++
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

	var got []string
	cfg.DeepEach(func(c *Config) {
		got = append(got, fmt.Sprintf("%s %s", strings.Join(c.Path, "."), c.Version))
	})
	sort.Strings(got)
	want := []string{
		" <nil>",
		"child_a 1.0.0",
		"child_a.child_c 1.0.1",
		"child_b 1.0.2",
		"child_b.child_c 1.0.3",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("wrong result\ngot: %swant: %s", spew.Sdump(got), spew.Sdump(want))
	}

	if _, exists := cfg.Children["child_a"].Children["child_c"].Module.Outputs["hello"]; !exists {
		t.Fatalf("missing output 'hello' in child_a.child_c")
	}
	if _, exists := cfg.Children["child_b"].Children["child_c"].Module.Outputs["hello"]; !exists {
		t.Fatalf("missing output 'hello' in child_b.child_c")
	}
	if cfg.Children["child_a"].Children["child_c"].Module == cfg.Children["child_b"].Children["child_c"].Module {
		t.Fatalf("child_a.child_c is same object as child_b.child_c; should not be")
	}
}

func TestBuildConfigDiags(t *testing.T) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDir("testdata/nested-errors")
	assertNoDiagnostics(t, diags)
	if mod == nil {
		t.Fatal("got nil root module; want non-nil")
	}

	versionI := 0
	cfg, diags := BuildConfig(mod, ModuleWalkerFunc(
		func(req *ModuleRequest) (*Module, *version.Version, hcl.Diagnostics) {
			// For the sake of this test we're going to just treat our
			// SourceAddr as a path relative to our fixture directory.
			// A "real" implementation of ModuleWalker should accept the
			// various different source address syntaxes Terraform supports.
			sourcePath := filepath.Join("testdata/nested-errors", req.SourceAddr.String())

			mod, diags := parser.LoadConfigDir(sourcePath)
			version, _ := version.NewVersion(fmt.Sprintf("1.0.%d", versionI))
			versionI++
			return mod, version, diags
		}),
		MockDataLoaderFunc(func(provider *Provider) (*MockData, hcl.Diagnostics) {
			return nil, nil
		}),
	)

	wantDiag := `testdata/nested-errors/child_c/child_c.tf:5,1-8: ` +
		`Unsupported block type; Blocks of type "invalid" are not expected here.`
	assertExactDiagnostics(t, diags, []string{wantDiag})

	// we should still have module structure loaded
	var got []string
	cfg.DeepEach(func(c *Config) {
		got = append(got, fmt.Sprintf("%s %s", strings.Join(c.Path, "."), c.Version))
	})
	sort.Strings(got)
	want := []string{
		" <nil>",
		"child_a 1.0.0",
		"child_a.child_c 1.0.1",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("wrong result\ngot: %swant: %s", spew.Sdump(got), spew.Sdump(want))
	}
}

func TestBuildConfigChildModuleBackend(t *testing.T) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDir("testdata/nested-backend-warning")
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
			sourcePath := filepath.Join("testdata/nested-backend-warning", req.SourceAddr.String())

			mod, diags := parser.LoadConfigDir(sourcePath)
			version, _ := version.NewVersion("1.0.0")
			return mod, version, diags
		}),
		MockDataLoaderFunc(func(provider *Provider) (*MockData, hcl.Diagnostics) {
			return nil, nil
		}),
	)

	assertDiagnosticSummary(t, diags, "Backend configuration ignored")

	// we should still have module structure loaded
	var got []string
	cfg.DeepEach(func(c *Config) {
		got = append(got, fmt.Sprintf("%s %s", strings.Join(c.Path, "."), c.Version))
	})
	sort.Strings(got)
	want := []string{
		" <nil>",
		"child 1.0.0",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("wrong result\ngot: %swant: %s", spew.Sdump(got), spew.Sdump(want))
	}
}

func TestBuildConfigInvalidModules(t *testing.T) {
	testDir := "testdata/config-diagnostics"
	dirs, err := ioutil.ReadDir(testDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range dirs {
		name := info.Name()
		t.Run(name, func(t *testing.T) {
			parser := NewParser(nil)
			path := filepath.Join(testDir, name)

			mod, diags := parser.LoadConfigDirWithTests(path, "tests")
			if diags.HasErrors() {
				// these tests should only trigger errors that are caught in
				// the config loader.
				t.Errorf("error loading config dir")
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
			}

			readDiags := func(data []byte, _ error) []string {
				var expected []string
				for _, s := range strings.Split(string(data), "\n") {
					msg := strings.TrimSpace(s)
					msg = strings.ReplaceAll(msg, `\n`, "\n")
					if msg != "" {
						expected = append(expected, msg)
					}
				}
				return expected
			}

			// Load expected errors and warnings.
			// Each line in the file is matched as a substring against the
			// diagnostic outputs.
			// Capturing part of the path and source range in the message lets
			// us also ensure the diagnostic is being attributed to the
			// expected location in the source, but is not required.
			// The literal characters `\n` are replaced with newlines, but
			// otherwise the string is unchanged.
			expectedErrs := readDiags(ioutil.ReadFile(filepath.Join(testDir, name, "errors")))
			expectedWarnings := readDiags(ioutil.ReadFile(filepath.Join(testDir, name, "warnings")))

			_, buildDiags := BuildConfig(mod, ModuleWalkerFunc(
				func(req *ModuleRequest) (*Module, *version.Version, hcl.Diagnostics) {
					// for simplicity, these tests will treat all source
					// addresses as relative to the root module
					sourcePath := filepath.Join(path, req.SourceAddr.String())
					mod, diags := parser.LoadConfigDir(sourcePath)
					version, _ := version.NewVersion("1.0.0")
					return mod, version, diags
				}),
				MockDataLoaderFunc(func(provider *Provider) (*MockData, hcl.Diagnostics) {
					return nil, nil
				}),
			)

			// we can make this less repetitive later if we want
			for _, msg := range expectedErrs {
				found := false
				for _, diag := range buildDiags {
					if diag.Severity == hcl.DiagError && strings.Contains(diag.Error(), msg) {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected error diagnostic containing:\n    %s", msg)
				}
			}

			for _, diag := range buildDiags {
				if diag.Severity != hcl.DiagError {
					continue
				}
				found := false
				for _, msg := range expectedErrs {
					if strings.Contains(diag.Error(), msg) {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Unexpected error:\n    %s", diag)
				}
			}

			for _, msg := range expectedWarnings {
				found := false
				for _, diag := range buildDiags {
					if diag.Severity == hcl.DiagWarning && strings.Contains(diag.Error(), msg) {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected warning diagnostic containing:\n    %s", msg)
				}
			}

			for _, diag := range buildDiags {
				if diag.Severity != hcl.DiagWarning {
					continue
				}
				found := false
				for _, msg := range expectedWarnings {
					if strings.Contains(diag.Error(), msg) {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Unexpected warning:\n    %s", diag)
				}
			}

		})
	}
}

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

	provider := cfg.Module.Tests["main.tftest.hcl"].Providers["aws"]

	// This time we want to check that the mock data defined inline took
	// precedence over the mock data defined in the data files.
	defaults := provider.MockData.MockResources["aws_s3_bucket"].Defaults
	expected := cty.ObjectVal(map[string]cty.Value{
		"arn": cty.StringVal("aws:s3:::bucket"),
	})

	if !defaults.RawEquals(expected) {
		t.Errorf("expected: %s\nactual:   %s", expected.GoString(), defaults.GoString())
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

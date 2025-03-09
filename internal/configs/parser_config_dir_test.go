// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
)

// TestParseLoadConfigDirSuccess is a simple test that just verifies that
// a number of test configuration directories (in testdata/valid-modules)
// can be parsed without raising any diagnostics.
//
// It also re-tests the individual files in testdata/valid-files as if
// they were single-file modules, to ensure that they can be bundled into
// modules correctly.
//
// This test does not verify that reading these modules produces the correct
// module element contents. More detailed assertions may be made on some subset
// of these configuration files in other tests.
func TestParserLoadConfigDirSuccess(t *testing.T) {
	dirs, err := ioutil.ReadDir("testdata/valid-modules")
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range dirs {
		name := info.Name()
		t.Run(name, func(t *testing.T) {
			parser := NewParser(nil)
			path := filepath.Join("testdata/valid-modules", name)

			mod, diags := parser.LoadConfigDir(path)
			if len(diags) != 0 && len(mod.ActiveExperiments) != 0 {
				// As a special case to reduce churn while we're working
				// through experimental features, we'll ignore the warning
				// that an experimental feature is active if the module
				// intentionally opted in to that feature.
				// If you want to explicitly test for the feature warning
				// to be generated, consider using testdata/warning-files
				// instead.
				filterDiags := make(hcl.Diagnostics, 0, len(diags))
				for _, diag := range diags {
					if diag.Severity != hcl.DiagWarning {
						continue
					}
					match := false
					for exp := range mod.ActiveExperiments {
						allowedSummary := fmt.Sprintf("Experimental feature %q is active", exp.Keyword())
						if diag.Summary == allowedSummary {
							match = true
							break
						}
					}
					if !match {
						filterDiags = append(filterDiags, diag)
					}
				}
				diags = filterDiags
			}
			if len(diags) != 0 {
				t.Errorf("unexpected diagnostics")
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
			}

			if mod.SourceDir != path {
				t.Errorf("wrong SourceDir value %q; want %s", mod.SourceDir, path)
			}

			if len(mod.Tests) > 0 {
				// We only load tests when requested, and we didn't request this
				// time.
				t.Errorf("should not have loaded tests, but found %d", len(mod.Tests))
			}
		})
	}

	// The individual files in testdata/valid-files should also work
	// when loaded as modules.
	files, err := ioutil.ReadDir("testdata/valid-files")
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range files {
		name := info.Name()
		t.Run(fmt.Sprintf("%s as module", name), func(t *testing.T) {
			src, err := ioutil.ReadFile(filepath.Join("testdata/valid-files", name))
			if err != nil {
				t.Fatal(err)
			}

			parser := testParser(map[string]string{
				"mod/" + name: string(src),
			})

			_, diags := parser.LoadConfigDir("mod")
			if diags.HasErrors() {
				t.Errorf("unexpected error diagnostics")
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
			}
		})
	}

}

func TestParserLoadConfigDirWithTests(t *testing.T) {
	directories := []string{
		"testdata/valid-modules/with-tests",
		"testdata/valid-modules/with-tests-expect-failures",
		"testdata/valid-modules/with-tests-nested",
		"testdata/valid-modules/with-tests-very-nested",
		"testdata/valid-modules/with-tests-json",
		"testdata/valid-modules/with-mocks",
	}

	for _, directory := range directories {
		t.Run(directory, func(t *testing.T) {

			testDirectory := DefaultTestDirectory
			if directory == "testdata/valid-modules/with-tests-very-nested" {
				testDirectory = "very/nested"
			}

			parser := NewParser(nil)
			mod, diags := parser.LoadConfigDirWithTests(directory, testDirectory)
			if len(diags) > 0 { // We don't want any warnings or errors.
				t.Errorf("unexpected diagnostics")
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
			}

			if len(mod.Tests) != 2 {
				t.Errorf("incorrect number of test files found: %d", len(mod.Tests))
			}
		})
	}
}

func TestParserLoadTestFiles_Invalid(t *testing.T) {

	tcs := map[string][]string{
		"duplicate_data_overrides": {
			"duplicate_data_overrides.tftest.hcl:7,3-16: Duplicate override_data block; An override_data block targeting data.aws_instance.test has already been defined at duplicate_data_overrides.tftest.hcl:2,3-16.",
			"duplicate_data_overrides.tftest.hcl:18,1-14: Duplicate override_data block; An override_data block targeting data.aws_instance.test has already been defined at duplicate_data_overrides.tftest.hcl:13,1-14.",
			"duplicate_data_overrides.tftest.hcl:29,3-16: Duplicate override_data block; An override_data block targeting data.aws_instance.test has already been defined at duplicate_data_overrides.tftest.hcl:24,3-16.",
		},
		"duplicate_mixed_providers": {
			"duplicate_mixed_providers.tftest.hcl:3,1-20: Duplicate provider block; A provider for aws is already defined at duplicate_mixed_providers.tftest.hcl:1,10-15.",
			"duplicate_mixed_providers.tftest.hcl:9,1-20: Duplicate provider block; A provider for aws.test is already defined at duplicate_mixed_providers.tftest.hcl:5,10-15.",
		},
		"duplicate_mock_data_sources": {
			"duplicate_mock_data_sources.tftest.hcl:7,13-27: Duplicate mock_data block; A mock_data block for aws_instance has already been defined at duplicate_mock_data_sources.tftest.hcl:3,3-27.",
		},
		"duplicate_mock_providers": {
			"duplicate_mock_providers.tftest.hcl:3,1-20: Duplicate provider block; A provider for aws is already defined at duplicate_mock_providers.tftest.hcl:1,15-20.",
			"duplicate_mock_providers.tftest.hcl:9,1-20: Duplicate provider block; A provider for aws.test is already defined at duplicate_mock_providers.tftest.hcl:5,15-20.",
		},
		"duplicate_mock_resources": {
			"duplicate_mock_resources.tftest.hcl:7,17-31: Duplicate mock_resource block; A mock_resource block for aws_instance has already been defined at duplicate_mock_resources.tftest.hcl:3,3-31.",
		},
		"duplicate_module_overrides": {
			"duplicate_module_overrides.tftest.hcl:7,1-16: Duplicate override_module block; An override_module block targeting module.child has already been defined at duplicate_module_overrides.tftest.hcl:2,1-16.",
			"duplicate_module_overrides.tftest.hcl:18,3-18: Duplicate override_module block; An override_module block targeting module.child has already been defined at duplicate_module_overrides.tftest.hcl:13,3-18.",
		},
		"duplicate_providers": {
			"duplicate_providers.tftest.hcl:3,1-15: Duplicate provider block; A provider for aws is already defined at duplicate_providers.tftest.hcl:1,10-15.",
			"duplicate_providers.tftest.hcl:9,1-15: Duplicate provider block; A provider for aws.test is already defined at duplicate_providers.tftest.hcl:5,10-15.",
		},
		"duplicate_resource_overrides": {
			"duplicate_resource_overrides.tftest.hcl:7,3-20: Duplicate override_resource block; An override_resource block targeting aws_instance.test has already been defined at duplicate_resource_overrides.tftest.hcl:2,3-20.",
			"duplicate_resource_overrides.tftest.hcl:18,1-18: Duplicate override_resource block; An override_resource block targeting aws_instance.test has already been defined at duplicate_resource_overrides.tftest.hcl:13,1-18.",
			"duplicate_resource_overrides.tftest.hcl:29,3-20: Duplicate override_resource block; An override_resource block targeting aws_instance.test has already been defined at duplicate_resource_overrides.tftest.hcl:24,3-20.",
		},
		"invalid_data_override": {
			"invalid_data_override.tftest.hcl:6,1-14: Missing target attribute; override_data blocks must specify a target address.",
		},
		"invalid_data_override_target": {
			"invalid_data_override_target.tftest.hcl:8,3-24: Invalid override target; You can only target data sources from override_data blocks, not module.child.",
			"invalid_data_override_target.tftest.hcl:3,3-31: Invalid override target; You can only target data sources from override_data blocks, not aws_instance.target.",
		},
		"invalid_mock_data_sources": {
			"invalid_mock_data_sources.tftest.hcl:7,13-16: Variables not allowed; Variables may not be used here.",
		},
		"invalid_mock_resources": {
			"invalid_mock_resources.tftest.hcl:7,13-16: Variables not allowed; Variables may not be used here.",
		},
		"invalid_module_override": {
			"invalid_module_override.tftest.hcl:5,1-16: Missing target attribute; override_module blocks must specify a target address.",
			"invalid_module_override.tftest.hcl:11,3-9: Unsupported argument; An argument named \"values\" is not expected here.",
		},
		"invalid_module_override_target": {
			"invalid_module_override_target.tftest.hcl:3,3-31: Invalid override target; You can only target modules from override_module blocks, not aws_instance.target.",
			"invalid_module_override_target.tftest.hcl:8,3-36: Invalid override target; You can only target modules from override_module blocks, not data.aws_instance.target.",
		},
		"invalid_resource_override": {
			"invalid_resource_override.tftest.hcl:6,1-18: Missing target attribute; override_resource blocks must specify a target address.",
		},
		"invalid_resource_override_target": {
			"invalid_resource_override_target.tftest.hcl:3,3-36: Invalid override target; You can only target resources from override_resource blocks, not data.aws_instance.target.",
			"invalid_resource_override_target.tftest.hcl:8,3-24: Invalid override target; You can only target resources from override_resource blocks, not module.child.",
		},
		"duplicate_file_config": {
			"duplicate_file_config.tftest.hcl:3,1-5: Multiple \"test\" blocks; This test file already has a \"test\" block defined at duplicate_file_config.tftest.hcl:1,1-5.",
			"duplicate_file_config.tftest.hcl:5,1-5: Multiple \"test\" blocks; This test file already has a \"test\" block defined at duplicate_file_config.tftest.hcl:1,1-5.",
		},
	}

	for name, expected := range tcs {
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(fmt.Sprintf("testdata/invalid-test-files/%s.tftest.hcl", name))
			if err != nil {
				t.Fatal(err)
			}

			parser := testParser(map[string]string{
				fmt.Sprintf("%s.tftest.hcl", name): string(src),
			})

			_, actual := parser.LoadTestFile(fmt.Sprintf("%s.tftest.hcl", name))
			assertExactDiagnostics(t, actual, expected)
		})
	}
}

func TestParserLoadConfigDirWithTests_ReturnsWarnings(t *testing.T) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDirWithTests("testdata/valid-modules/with-tests", "not_real")
	if len(diags) != 1 {
		t.Errorf("expected exactly 1 diagnostic, but found %d", len(diags))
	} else {
		if diags[0].Severity != hcl.DiagWarning {
			t.Errorf("expected warning severity but found %d", diags[0].Severity)
		}

		if diags[0].Summary != "Test directory does not exist" {
			t.Errorf("expected summary to be \"Test directory does not exist\" but was \"%s\"", diags[0].Summary)
		}

		if diags[0].Detail != "Requested test directory testdata/valid-modules/with-tests/not_real does not exist." {
			t.Errorf("expected detail to be \"Requested test directory testdata/valid-modules/with-tests/not_real does not exist.\" but was \"%s\"", diags[0].Detail)
		}
	}

	// Despite the warning, should still have loaded the tests in the
	// configuration directory.
	if len(mod.Tests) != 2 {
		t.Errorf("incorrect number of test files found: %d", len(mod.Tests))
	}
}

// TestParseLoadConfigDirFailure is a simple test that just verifies that
// a number of test configuration directories (in testdata/invalid-modules)
// produce diagnostics when parsed.
//
// It also re-tests the individual files in testdata/invalid-files as if
// they were single-file modules, to ensure that their errors are still
// detected when loading as part of a module.
//
// This test does not verify that reading these modules produces any
// diagnostics in particular. More detailed assertions may be made on some subset
// of these configuration files in other tests.
func TestParserLoadConfigDirFailure(t *testing.T) {
	dirs, err := ioutil.ReadDir("testdata/invalid-modules")
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range dirs {
		name := info.Name()
		t.Run(name, func(t *testing.T) {
			parser := NewParser(nil)
			path := filepath.Join("testdata/invalid-modules", name)

			_, diags := parser.LoadConfigDirWithTests(path, "tests")
			if !diags.HasErrors() {
				t.Errorf("no errors; want at least one")
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
			}
		})
	}

	// The individual files in testdata/valid-files should also work
	// when loaded as modules.
	files, err := ioutil.ReadDir("testdata/invalid-files")
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range files {
		name := info.Name()
		t.Run(fmt.Sprintf("%s as module", name), func(t *testing.T) {
			src, err := ioutil.ReadFile(filepath.Join("testdata/invalid-files", name))
			if err != nil {
				t.Fatal(err)
			}

			parser := testParser(map[string]string{
				"mod/" + name: string(src),
			})

			_, diags := parser.LoadConfigDir("mod")
			if !diags.HasErrors() {
				t.Errorf("no errors; want at least one")
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
			}
		})
	}

}

func TestIsEmptyDir(t *testing.T) {
	val, err := IsEmptyDir(filepath.Join("testdata", "valid-files"), "")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if val {
		t.Fatal("should not be empty")
	}
}

func TestIsEmptyDir_noExist(t *testing.T) {
	val, err := IsEmptyDir(filepath.Join("testdata", "nopenopenope"), "")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !val {
		t.Fatal("should be empty")
	}
}

func TestIsEmptyDir_noConfigsAndTests(t *testing.T) {
	val, err := IsEmptyDir(filepath.Join("testdata", "dir-empty"), "")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !val {
		t.Fatal("should be empty")
	}
}

func TestIsEmptyDir_noConfigsButHasTests(t *testing.T) {
	// The top directory has no configs, but it contains test files
	val, err := IsEmptyDir(filepath.Join("testdata", "only-test-files"), "tests")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if val {
		t.Fatal("should not be empty")
	}
}

func TestIsEmptyDir_nestedTestsOnly(t *testing.T) {
	// The top directory has no configs and no test files, but the nested
	// directory has test files
	val, err := IsEmptyDir(filepath.Join("testdata", "only-nested-test-files"), "tests")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if val {
		t.Fatal("should not be empty")
	}
}

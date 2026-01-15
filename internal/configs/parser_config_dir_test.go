// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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

			if strings.Contains(name, "state-store") {
				// The PSS project is currently gated as experimental
				// TODO(SarahFrench/radeksimko) - remove this from the test once
				// the feature is GA.
				parser.allowExperiments = true
			}

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
		"testdata/valid-modules/with-tests-backend",
		"testdata/valid-modules/with-tests-same-backend-across-files",
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
			parser.AllowLanguageExperiments(true)
			mod, diags := parser.LoadConfigDir(directory, MatchTestFiles(testDirectory))
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

func TestParserLoadConfigDirWithQueries(t *testing.T) {
	tests := []struct {
		name             string
		directory        string
		diagnostics      []string
		listResources    int
		managedResources int
	}{
		{
			name:          "simple",
			directory:     "testdata/query-files/valid/simple",
			listResources: 3,
		},
		{
			name:             "mixed",
			directory:        "testdata/query-files/valid/mixed",
			listResources:    3,
			managedResources: 1,
		},
		{
			name:             "loading query lists with no-experiments",
			directory:        "testdata/query-files/valid/mixed",
			managedResources: 1,
			listResources:    3,
		},
		{
			name:      "no-provider",
			directory: "testdata/query-files/invalid/no-provider",
			diagnostics: []string{
				"testdata/query-files/invalid/no-provider/main.tfquery.hcl:1,1-27: Missing \"provider\" attribute; You must specify a provider attribute when defining a list block.",
			},
		},
		{
			name:      "with-depends-on",
			directory: "testdata/query-files/invalid/with-depends-on",
			diagnostics: []string{
				"testdata/query-files/invalid/with-depends-on/main.tfquery.hcl:23,3-13: Unsupported argument; An argument named \"depends_on\" is not expected here.",
			},
			listResources: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parser := NewParser(nil)
			mod, diags := parser.LoadConfigDir(test.directory, MatchQueryFiles())
			if len(test.diagnostics) > 0 {
				if !diags.HasErrors() {
					t.Errorf("expected errors, but found none")
				}
				if len(diags) != len(test.diagnostics) {
					t.Fatalf("expected %d errors, but found %d", len(test.diagnostics), len(diags))
				}
				for i, diag := range diags {
					if diag.Error() != test.diagnostics[i] {
						t.Errorf("expected error to be %q, but found %q", test.diagnostics[i], diag.Error())
					}
				}
			} else {
				if len(diags) > 0 { // We don't want any warnings or errors.
					t.Errorf("unexpected diagnostics")
					for _, diag := range diags {
						t.Logf("- %s", diag)
					}
				}
			}

			if len(mod.ListResources) != test.listResources {
				t.Errorf("incorrect number of list blocks found: %d", len(mod.ListResources))
			}

			if len(mod.ManagedResources) != test.managedResources {
				t.Errorf("incorrect number of managed blocks found: %d", len(mod.ManagedResources))
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
		"duplicate_backend_blocks_in_test": {
			"duplicate_backend_blocks_in_test.tftest.hcl:15,3-18: Duplicate backend blocks; The run \"test\" already uses an internal state file that's loaded by a backend in the run \"setup\". Please ensure that a backend block is only in the first apply run block for a given internal state file.",
		},
		"duplicate_backend_blocks_in_run": {
			"duplicate_backend_blocks_in_run.tftest.hcl:6,3-18: Duplicate backend blocks; A backend block has already been defined inside the run \"setup\" at duplicate_backend_blocks_in_run.tftest.hcl:3,3-18.",
		},
		"backend_block_in_plan_run": {
			"backend_block_in_plan_run.tftest.hcl:6,3-18: Invalid backend block; A backend block can only be used in the first apply run block for a given internal state file. It cannot be included in a block to run a plan command.",
		},
		"backend_block_in_second_apply_run": {
			"backend_block_in_second_apply_run.tftest.hcl:10,3-18: Invalid backend block; The run \"test_2\" cannot load in state using a backend block, because internal state has already been created by an apply command in run \"test_1\". Backend blocks can only be present in the first apply command for a given internal state.",
		},
		"non_state_storage_backend_in_test": {
			"non_state_storage_backend_in_test.tftest.hcl:4,3-19: Invalid backend block; The \"remote\" backend type cannot be used in the backend block in run \"test\" at non_state_storage_backend_in_test.tftest.hcl:4,3-19. Only state storage backends can be used in a test run.",
		},
		"skip_cleanup_after_backend": {
			"skip_cleanup_after_backend.tftest.hcl:13,3-15: Duplicate \"skip_cleanup\" block; The run \"skip_cleanup\" has a skip_cleanup attribute set, but shares state with an earlier run \"backend\" that has a backend defined. The later run takes precedence, but the backend will still be used to manage this state.",
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
			parser.AllowLanguageExperiments(true)

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

		if !strings.HasPrefix(diags[0].Detail, "Requested test directory testdata/valid-modules/with-tests/not_real does not exist.") {
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

			_, diags := parser.LoadConfigDir(path, MatchTestFiles("tests"))
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

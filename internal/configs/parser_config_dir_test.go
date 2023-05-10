// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"fmt"
	"io/ioutil"
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

			_, diags := parser.LoadConfigDir(path)
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
	val, err := IsEmptyDir(filepath.Join("testdata", "valid-files"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if val {
		t.Fatal("should not be empty")
	}
}

func TestIsEmptyDir_noExist(t *testing.T) {
	val, err := IsEmptyDir(filepath.Join("testdata", "nopenopenope"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !val {
		t.Fatal("should be empty")
	}
}

func TestIsEmptyDir_noConfigs(t *testing.T) {
	val, err := IsEmptyDir(filepath.Join("testdata", "dir-empty"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !val {
		t.Fatal("should be empty")
	}
}

package configs

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
)

// TestParseLoadConfigDirSuccess is a simple test that just verifies that
// a number of test configuration directories (in test-fixtures/valid-modules)
// can be parsed without raising any diagnostics.
//
// It also re-tests the individual files in test-fixtures/valid-files as if
// they were single-file modules, to ensure that they can be bundled into
// modules correctly.
//
// This test does not verify that reading these modules produces the correct
// module element contents. More detailed assertions may be made on some subset
// of these configuration files in other tests.
func TestParserLoadConfigDirSuccess(t *testing.T) {
	dirs, err := ioutil.ReadDir("test-fixtures/valid-modules")
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range dirs {
		name := info.Name()
		t.Run(name, func(t *testing.T) {
			parser := NewParser(nil)
			path := filepath.Join("test-fixtures/valid-modules", name)

			_, diags := parser.LoadConfigDir(path)
			if len(diags) != 0 {
				t.Errorf("unexpected diagnostics")
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
			}
		})
	}

	// The individual files in test-fixtures/valid-files should also work
	// when loaded as modules.
	files, err := ioutil.ReadDir("test-fixtures/valid-files")
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range files {
		name := info.Name()
		t.Run(fmt.Sprintf("%s as module", name), func(t *testing.T) {
			src, err := ioutil.ReadFile(filepath.Join("test-fixtures/valid-files", name))
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
// a number of test configuration directories (in test-fixtures/invalid-modules)
// produce diagnostics when parsed.
//
// It also re-tests the individual files in test-fixtures/invalid-files as if
// they were single-file modules, to ensure that their errors are still
// detected when loading as part of a module.
//
// This test does not verify that reading these modules produces any
// diagnostics in particular. More detailed assertions may be made on some subset
// of these configuration files in other tests.
func TestParserLoadConfigDirFailure(t *testing.T) {
	dirs, err := ioutil.ReadDir("test-fixtures/invalid-modules")
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range dirs {
		name := info.Name()
		t.Run(name, func(t *testing.T) {
			parser := NewParser(nil)
			path := filepath.Join("test-fixtures/invalid-modules", name)

			_, diags := parser.LoadConfigDir(path)
			if !diags.HasErrors() {
				t.Errorf("no errors; want at least one")
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
			}
		})
	}

	// The individual files in test-fixtures/valid-files should also work
	// when loaded as modules.
	files, err := ioutil.ReadDir("test-fixtures/invalid-files")
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range files {
		name := info.Name()
		t.Run(fmt.Sprintf("%s as module", name), func(t *testing.T) {
			src, err := ioutil.ReadFile(filepath.Join("test-fixtures/invalid-files", name))
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

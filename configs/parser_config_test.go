package configs

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl2/hcl"
)

// TestParseLoadConfigFileSuccess is a simple test that just verifies that
// a number of test configuration files (in test-fixtures/valid-files) can
// be parsed without raising any diagnostics.
//
// This test does not verify that reading these files produces the correct
// file element contents. More detailed assertions may be made on some subset
// of these configuration files in other tests.
func TestParserLoadConfigFileSuccess(t *testing.T) {
	files, err := ioutil.ReadDir("test-fixtures/valid-files")
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range files {
		name := info.Name()
		t.Run(name, func(t *testing.T) {
			src, err := ioutil.ReadFile(filepath.Join("test-fixtures/valid-files", name))
			if err != nil {
				t.Fatal(err)
			}

			parser := testParser(map[string]string{
				name: string(src),
			})

			_, diags := parser.LoadConfigFile(name)
			if len(diags) != 0 {
				t.Errorf("unexpected diagnostics")
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
			}
		})
	}
}

// TestParseLoadConfigFileFailure is a simple test that just verifies that
// a number of test configuration files (in test-fixtures/invalid-files)
// produce errors as expected.
//
// This test does not verify specific error messages, so more detailed
// assertions should be made on some subset of these configuration files in
// other tests.
func TestParserLoadConfigFileFailure(t *testing.T) {
	files, err := ioutil.ReadDir("test-fixtures/invalid-files")
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range files {
		name := info.Name()
		t.Run(name, func(t *testing.T) {
			src, err := ioutil.ReadFile(filepath.Join("test-fixtures/invalid-files", name))
			if err != nil {
				t.Fatal(err)
			}

			parser := testParser(map[string]string{
				name: string(src),
			})

			_, diags := parser.LoadConfigFile(name)
			if !diags.HasErrors() {
				t.Errorf("LoadConfigFile succeeded; want errors")
			}
			for _, diag := range diags {
				t.Logf("- %s", diag)
			}
		})
	}
}

// This test uses a subset of the same fixture files as
// TestParserLoadConfigFileFailure, but additionally verifies that each
// file produces the expected diagnostic summary.
func TestParserLoadConfigFileFailureMessages(t *testing.T) {
	tests := []struct {
		Filename  string
		WantError string
	}{
		{
			"data-resource-lifecycle.tf",
			"Unsupported lifecycle block",
		},
		{
			"variable-type-unknown.tf",
			"Invalid variable type hint",
		},
		{
			"variable-type-quoted.tf",
			"Invalid variable type hint",
		},
		{
			"unexpected-attr.tf",
			"Unsupported attribute",
		},
		{
			"unexpected-block.tf",
			"Unsupported block type",
		},
		{
			"resource-lifecycle-badbool.tf",
			"Unsuitable value type",
		},
	}

	for _, test := range tests {
		t.Run(test.Filename, func(t *testing.T) {
			src, err := ioutil.ReadFile(filepath.Join("test-fixtures/invalid-files", test.Filename))
			if err != nil {
				t.Fatal(err)
			}

			parser := testParser(map[string]string{
				test.Filename: string(src),
			})

			_, diags := parser.LoadConfigFile(test.Filename)
			if len(diags) != 1 {
				t.Errorf("Wrong number of diagnostics %d; want 1", len(diags))
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
				return
			}
			if diags[0].Severity != hcl.DiagError {
				t.Errorf("Wrong diagnostic severity %s; want %s", diags[0].Severity, hcl.DiagError)
			}
			if diags[0].Summary != test.WantError {
				t.Errorf("Wrong diagnostic summary\ngot:  %s\nwant: %s", diags[0].Summary, test.WantError)
			}
		})
	}
}

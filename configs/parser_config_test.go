package configs

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl2/hcl"
)

// TestParseLoadConfigFileSuccess is a simple test that just verifies that
// a number of test configuration files (in testdata/valid-files) can
// be parsed without raising any diagnostics.
//
// This test does not verify that reading these files produces the correct
// file element contents. More detailed assertions may be made on some subset
// of these configuration files in other tests.
func TestParserLoadConfigFileSuccess(t *testing.T) {
	files, err := ioutil.ReadDir("testdata/valid-files")
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range files {
		name := info.Name()
		t.Run(name, func(t *testing.T) {
			src, err := ioutil.ReadFile(filepath.Join("testdata/valid-files", name))
			if err != nil {
				t.Fatal(err)
			}

			parser := testParser(map[string]string{
				name: string(src),
			})

			_, diags := parser.LoadConfigFile(name)
			if diags.HasErrors() {
				t.Errorf("unexpected error diagnostics")
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
			}
		})
	}
}

// TestParseLoadConfigFileFailure is a simple test that just verifies that
// a number of test configuration files (in testdata/invalid-files)
// produce errors as expected.
//
// This test does not verify specific error messages, so more detailed
// assertions should be made on some subset of these configuration files in
// other tests.
func TestParserLoadConfigFileFailure(t *testing.T) {
	files, err := ioutil.ReadDir("testdata/invalid-files")
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range files {
		name := info.Name()
		t.Run(name, func(t *testing.T) {
			src, err := ioutil.ReadFile(filepath.Join("testdata/invalid-files", name))
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
		Filename     string
		WantSeverity hcl.DiagnosticSeverity
		WantDiag     string
	}{
		{
			"invalid-files/data-resource-lifecycle.tf",
			hcl.DiagError,
			"Unsupported lifecycle block",
		},
		{
			"invalid-files/variable-type-unknown.tf",
			hcl.DiagError,
			"Invalid type specification",
		},
		{
			"invalid-files/unexpected-attr.tf",
			hcl.DiagError,
			"Unsupported argument",
		},
		{
			"invalid-files/unexpected-block.tf",
			hcl.DiagError,
			"Unsupported block type",
		},
		{
			"invalid-files/resource-count-and-for_each.tf",
			hcl.DiagError,
			`Invalid combination of "count" and "for_each"`,
		},
		{
			"invalid-files/data-count-and-for_each.tf",
			hcl.DiagError,
			`Invalid combination of "count" and "for_each"`,
		},
		{
			"invalid-files/resource-lifecycle-badbool.tf",
			hcl.DiagError,
			"Unsuitable value type",
		},
		{
			"valid-files/resources-ignorechanges-all-legacy.tf",
			hcl.DiagWarning,
			"Deprecated ignore_changes wildcard",
		},
		{
			"valid-files/resources-ignorechanges-all-legacy.tf.json",
			hcl.DiagWarning,
			"Deprecated ignore_changes wildcard",
		},
	}

	for _, test := range tests {
		t.Run(test.Filename, func(t *testing.T) {
			src, err := ioutil.ReadFile(filepath.Join("testdata", test.Filename))
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
			if diags[0].Severity != test.WantSeverity {
				t.Errorf("Wrong diagnostic severity %#v; want %#v", diags[0].Severity, test.WantSeverity)
			}
			if diags[0].Summary != test.WantDiag {
				t.Errorf("Wrong diagnostic summary\ngot:  %s\nwant: %s", diags[0].Summary, test.WantDiag)
			}
		})
	}
}

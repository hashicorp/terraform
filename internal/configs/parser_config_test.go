// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/hcl/v2"
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
			"Invalid data resource lifecycle argument",
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

// TestParseLoadConfigFileWarning is a test that verifies files from
// testdata/warning-files produce particular warnings.
//
// This test does not verify that reading these files produces the correct
// file element contents in spite of those warnings. More detailed assertions
// may be made on some subset of these configuration files in other tests.
func TestParserLoadConfigFileWarning(t *testing.T) {
	files, err := ioutil.ReadDir("testdata/warning-files")
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range files {
		name := info.Name()
		t.Run(name, func(t *testing.T) {
			src, err := ioutil.ReadFile(filepath.Join("testdata/warning-files", name))
			if err != nil {
				t.Fatal(err)
			}

			// First we'll scan the file to see what warnings are expected.
			// That's declared inside the files themselves by using the
			// string "WARNING: " somewhere on each line that is expected
			// to produce a warning, followed by the expected warning summary
			// text. A single-line comment (with #) is the main way to do that.
			const marker = "WARNING: "
			sc := bufio.NewScanner(bytes.NewReader(src))
			wantWarnings := make(map[int]string)
			lineNum := 1
			for sc.Scan() {
				lineText := sc.Text()
				if idx := strings.Index(lineText, marker); idx != -1 {
					summaryText := lineText[idx+len(marker):]
					wantWarnings[lineNum] = summaryText
				}
				lineNum++
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

			gotWarnings := make(map[int]string)
			for _, diag := range diags {
				if diag.Severity != hcl.DiagWarning || diag.Subject == nil {
					continue
				}
				gotWarnings[diag.Subject.Start.Line] = diag.Summary
			}

			if diff := cmp.Diff(wantWarnings, gotWarnings); diff != "" {
				t.Errorf("wrong warnings\n%s", diff)
			}
		})
	}
}

// TestParseLoadConfigFileError is a test that verifies files from
// testdata/warning-files produce particular errors.
//
// This test does not verify that reading these files produces the correct
// file element contents in spite of those errors. More detailed assertions
// may be made on some subset of these configuration files in other tests.
func TestParserLoadConfigFileError(t *testing.T) {
	files, err := ioutil.ReadDir("testdata/error-files")
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range files {
		name := info.Name()
		t.Run(name, func(t *testing.T) {
			src, err := ioutil.ReadFile(filepath.Join("testdata/error-files", name))
			if err != nil {
				t.Fatal(err)
			}

			// First we'll scan the file to see what warnings are expected.
			// That's declared inside the files themselves by using the
			// string "ERROR: " somewhere on each line that is expected
			// to produce a warning, followed by the expected warning summary
			// text. A single-line comment (with #) is the main way to do that.
			const marker = "ERROR: "
			sc := bufio.NewScanner(bytes.NewReader(src))
			wantErrors := make(map[int]string)
			lineNum := 1
			for sc.Scan() {
				lineText := sc.Text()
				if idx := strings.Index(lineText, marker); idx != -1 {
					summaryText := lineText[idx+len(marker):]
					wantErrors[lineNum] = summaryText
				}
				lineNum++
			}

			parser := testParser(map[string]string{
				name: string(src),
			})

			_, diags := parser.LoadConfigFile(name)

			gotErrors := make(map[int]string)
			for _, diag := range diags {
				if diag.Severity != hcl.DiagError || diag.Subject == nil {
					continue
				}
				gotErrors[diag.Subject.Start.Line] = diag.Summary
			}

			if diff := cmp.Diff(wantErrors, gotErrors); diff != "" {
				t.Errorf("wrong errors\n%s", diff)
			}
		})
	}
}

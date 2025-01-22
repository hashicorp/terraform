// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1
package junit

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func Test_TestJUnitXMLFile_save(t *testing.T) {

	cases := map[string]struct {
		filename    string
		expectError bool
	}{
		"can save output to the specified filename": {
			filename: func() string {
				td := t.TempDir()
				return fmt.Sprintf("%s/output.xml", td)
			}(),
		},
		"returns an error when given a filename that isn't absolute or relative": {
			filename:    "~/output.xml",
			expectError: true,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			j := TestJUnitXMLFile{
				filename: tc.filename,
			}

			xml := []byte(`<?xml version="1.0" encoding="UTF-8"?><testsuites>
  <testsuite name="example_1.tftest.hcl" tests="1" skipped="0" failures="0" errors="0">
    <testcase name="true_is_true" classname="example_1.tftest.hcl" time="0.005381209"></testcase>
  </testsuite>
</testsuites>`)

			diags := j.save(xml)

			if diags.HasErrors() {
				if !tc.expectError {
					t.Fatalf("got unexpected error: %s", diags.Err())
				}
				// return early if testing error case
				return
			}

			if !diags.HasErrors() && tc.expectError {
				t.Fatalf("expected an error but got none")
			}

			fileContent, err := os.ReadFile(tc.filename)
			if err != nil {
				t.Fatalf("unexpected error opening file")
			}

			if !bytes.Equal(fileContent, xml) {
				t.Fatalf("wanted XML:\n%s\n got XML:\n%s\n", string(xml), string(fileContent))
			}
		})
	}
}

func Test_getWarnings(t *testing.T) {
	errorDiag := &hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "error",
		Detail:   "this is an error",
	}

	warnDiag := &hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "warning",
		Detail:   "this is a warning",
	}

	cases := map[string]struct {
		diags    tfdiags.Diagnostics
		expected tfdiags.Diagnostics
	}{
		"empty diags": {
			diags:    tfdiags.Diagnostics{},
			expected: tfdiags.Diagnostics{},
		},
		"nil diags": {
			diags:    nil,
			expected: tfdiags.Diagnostics{},
		},
		"all error diags": {
			diags: func() tfdiags.Diagnostics {
				var d tfdiags.Diagnostics
				d = d.Append(errorDiag, errorDiag, errorDiag)
				return d
			}(),
			expected: tfdiags.Diagnostics{},
		},
		"mixture of error and warning diags": {
			diags: func() tfdiags.Diagnostics {
				var d tfdiags.Diagnostics
				d = d.Append(errorDiag, errorDiag, warnDiag) // 1 warning
				return d
			}(),
			expected: func() tfdiags.Diagnostics {
				var d tfdiags.Diagnostics
				d = d.Append(warnDiag) // 1 warning
				return d
			}(),
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			warnings := getWarnings(tc.diags)

			if diff := cmp.Diff(tc.expected, warnings, tfdiags.DiagnosticComparer); diff != "" {
				t.Errorf("wrong diagnostics\n%s", diff)
			}
		})
	}
}

func Test_suiteFilesAsSortedList(t *testing.T) {
	cases := map[string]struct {
		Suite         *moduletest.Suite
		ExpectedNames map[int]string
	}{
		"no test files": {
			Suite: &moduletest.Suite{},
		},
		"3 test files ordered in map": {
			Suite: &moduletest.Suite{
				Status: moduletest.Skip,
				Files: map[string]*moduletest.File{
					"test_file_1.tftest.hcl": {
						Name:   "test_file_1.tftest.hcl",
						Status: moduletest.Skip,
						Runs:   []*moduletest.Run{},
					},
					"test_file_2.tftest.hcl": {
						Name:   "test_file_2.tftest.hcl",
						Status: moduletest.Skip,
						Runs:   []*moduletest.Run{},
					},
					"test_file_3.tftest.hcl": {
						Name:   "test_file_3.tftest.hcl",
						Status: moduletest.Skip,
						Runs:   []*moduletest.Run{},
					},
				},
			},
			ExpectedNames: map[int]string{
				0: "test_file_1.tftest.hcl",
				1: "test_file_2.tftest.hcl",
				2: "test_file_3.tftest.hcl",
			},
		},
		"3 test files unordered in map": {
			Suite: &moduletest.Suite{
				Status: moduletest.Skip,
				Files: map[string]*moduletest.File{
					"test_file_3.tftest.hcl": {
						Name:   "test_file_3.tftest.hcl",
						Status: moduletest.Skip,
						Runs:   []*moduletest.Run{},
					},
					"test_file_1.tftest.hcl": {
						Name:   "test_file_1.tftest.hcl",
						Status: moduletest.Skip,
						Runs:   []*moduletest.Run{},
					},
					"test_file_2.tftest.hcl": {
						Name:   "test_file_2.tftest.hcl",
						Status: moduletest.Skip,
						Runs:   []*moduletest.Run{},
					},
				},
			},
			ExpectedNames: map[int]string{
				0: "test_file_1.tftest.hcl",
				1: "test_file_2.tftest.hcl",
				2: "test_file_3.tftest.hcl",
			},
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			list := suiteFilesAsSortedList(tc.Suite.Files)

			if len(tc.ExpectedNames) != len(tc.Suite.Files) {
				t.Fatalf("expected there to be %d items, got %d", len(tc.ExpectedNames), len(tc.Suite.Files))
			}

			if len(tc.ExpectedNames) == 0 {
				return
			}

			for k, v := range tc.ExpectedNames {
				if list[k].Name != v {
					t.Fatalf("expected element %d in sorted list to be named %s, got %s", k, v, list[k].Name)
				}
			}
		})
	}
}

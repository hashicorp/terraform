// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseGet_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Get
	}{
		"defaults": {
			nil,
			&Get{
				TestsDirectory: "tests",
			},
		},
		"all options": {
			[]string{
				"-update",
				"-test-directory=integration-tests",
				"path/to/module",
			},
			&Get{
				Update:         true,
				TestsDirectory: "integration-tests",
				Args:           []string{"path/to/module"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseGet(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseGet_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Get
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-wat"},
			&Get{
				TestsDirectory: "tests",
				Args:           []string{},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -wat",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseGet(tc.args)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

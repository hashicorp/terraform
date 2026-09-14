// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseMetadataFunctions_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *MetadataFunctions
	}{
		"json": {
			args: []string{"-json"},
			want: &MetadataFunctions{
				JSON: true,
			},
		},
		"json with ignored positional argument": {
			args: []string{"-json", "extra"},
			want: &MetadataFunctions{
				JSON: true,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseMetadataFunctions(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseMetadataFunctions_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *MetadataFunctions
		wantDiags tfdiags.Diagnostics
	}{
		"default values": {
			args: []string{},
			want: &MetadataFunctions{},
			wantDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"The -json flag is required",
					"The `terraform metadata functions` command requires the `-json` flag.",
				),
			},
		},
		"unknown flag": {
			args: []string{"-wat"},
			want: &MetadataFunctions{},
			wantDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -wat",
				),
			},
		},
		"unknown flag after json": {
			args: []string{"-json", "-wat"},
			want: &MetadataFunctions{
				JSON: true,
			},
			wantDiags: tfdiags.Diagnostics{
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
			got, gotDiags := ParseMetadataFunctions(tc.args)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseProvidersSchema_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *ProvidersSchema
	}{
		"json": {
			[]string{"-json"},
			&ProvidersSchema{
				JSON: true,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseProvidersSchema(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseProvidersSchema_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *ProvidersSchema
		wantDiags tfdiags.Diagnostics
	}{
		"missing json": {
			nil,
			&ProvidersSchema{},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"The -json flag is required",
					"The `terraform providers schema` command requires the `-json` flag.",
				),
			},
		},
		"too many positional arguments": {
			[]string{"-json", "extra"},
			&ProvidersSchema{
				JSON: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Too many command line arguments",
					"Expected no positional arguments.",
				),
			},
		},
		"unknown flag and missing json": {
			[]string{"-wat"},
			&ProvidersSchema{},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -wat",
				),
				tfdiags.Sourceless(
					tfdiags.Error,
					"The -json flag is required",
					"The `terraform providers schema` command requires the `-json` flag.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseProvidersSchema(tc.args)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

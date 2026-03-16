// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseProvidersMirror_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *ProvidersMirror
	}{
		"defaults": {
			[]string{"./mirror"},
			&ProvidersMirror{
				LockFile:  true,
				OutputDir: "./mirror",
			},
		},
		"all options": {
			[]string{
				"-platform=linux_amd64",
				"-platform=darwin_arm64",
				"-lock-file=false",
				"./mirror",
			},
			&ProvidersMirror{
				Platforms: FlagStringSlice{"linux_amd64", "darwin_arm64"},
				OutputDir: "./mirror",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseProvidersMirror(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseProvidersMirror_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *ProvidersMirror
		wantDiags tfdiags.Diagnostics
	}{
		"missing output directory": {
			nil,
			&ProvidersMirror{
				LockFile: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"No output directory specified",
					"The providers mirror command requires an output directory as a command-line argument.",
				),
			},
		},
		"too many arguments": {
			[]string{"./mirror", "./extra"},
			&ProvidersMirror{
				LockFile: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"No output directory specified",
					"The providers mirror command requires an output directory as a command-line argument.",
				),
			},
		},
		"unknown flag and missing output directory": {
			[]string{"-wat"},
			&ProvidersMirror{
				LockFile: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -wat",
				),
				tfdiags.Sourceless(
					tfdiags.Error,
					"No output directory specified",
					"The providers mirror command requires an output directory as a command-line argument.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseProvidersMirror(tc.args)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseProviders_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Providers
	}{
		"defaults": {
			nil,
			&Providers{
				Path:          ".",
				TestDirectory: "tests",
			},
		},
		"path": {
			[]string{"foo"},
			&Providers{
				Path:          "foo",
				TestDirectory: "tests",
			},
		},
		"test-directory": {
			[]string{"-test-directory", "other"},
			&Providers{
				Path:          ".",
				TestDirectory: "other",
			},
		},
		"path with test-directory": {
			[]string{"-test-directory", "other", "mypath"},
			&Providers{
				Path:          "mypath",
				TestDirectory: "other",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseProviders(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseProviders_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Providers
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop"},
			&Providers{
				Path:          ".",
				TestDirectory: "tests",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -boop",
				),
			},
		},
		"too many arguments": {
			[]string{"bar", "baz"},
			&Providers{
				Path:          "bar",
				TestDirectory: "tests",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Too many command line arguments",
					"Expected at most one positional argument.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseProviders(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

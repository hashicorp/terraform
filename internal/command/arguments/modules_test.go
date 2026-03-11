// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseModules_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Modules
	}{
		"default": {
			nil,
			&Modules{
				ViewType: ViewHuman,
			},
		},
		"json": {
			[]string{"-json"},
			&Modules{
				ViewType: ViewJSON,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseModules(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseModules_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Modules
		wantDiags tfdiags.Diagnostics
	}{
		"invalid flag": {
			[]string{"-sauron"},
			&Modules{
				ViewType: ViewHuman,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -sauron",
				),
			},
		},
		"too many arguments": {
			[]string{"-json", "frodo"},
			&Modules{
				ViewType: ViewJSON,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Too many command line arguments",
					"Expected no positional arguments",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseModules(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

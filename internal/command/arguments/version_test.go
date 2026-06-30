// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseVersion_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Version
	}{
		"defaults": {
			nil,
			&Version{
				ViewType: ViewHuman,
			},
		},
		"json": {
			[]string{"-json"},
			&Version{
				ViewType: ViewJSON,
			},
		},
		"too many arguments": { // Old behavior is to tolerate this, but we don't want to break it yet.
			[]string{"bar", "baz"},
			&Version{
				ViewType: ViewHuman,
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseVersion(tc.args, func() {
				t.Fatal("unexpected error parsing command flags")
			})
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseVersion_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Version
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop"},
			&Version{
				ViewType: ViewHuman,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -boop",
				),
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseVersion(tc.args, func() {})
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

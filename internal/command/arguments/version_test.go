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
		// User may run `terraform -v` which becomes `terraform version -v` due to command rerouting.
		"-v": {
			[]string{"-v"},
			&Version{
				ViewType: ViewHuman,
			},
		},
		// User may run `terraform -version` which becomes `terraform version -version` due to command rerouting.
		"-version": {
			[]string{"-version"},
			&Version{
				ViewType: ViewHuman,
			},
		},
		// User may run `terraform --version` which becomes `terraform version --version` due to command rerouting.
		"--version": {
			[]string{"--version"},
			&Version{
				ViewType: ViewHuman,
			},
		},
		// Old behavior we need to preserve (or address in calling code)
		// The version command could receive this if a user ran `terraform init -version -upgrade -get=false`
		// and the CLI rerouted it to the version command: `terraform version init -version -upgrade -get=false`
		"too many arguments, e.g. due to command rerouting": {
			[]string{"init", "-version", "-upgrade", "-get=false"},
			&Version{
				ViewType: ViewHuman,
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseVersion(tc.args)
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
		"unknown flag": { // This would happen specifically if the user supplied the non existent flag via `terraform -version -boop` or `terraform version -boop`.
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
			got, gotDiags := ParseVersion(tc.args)
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

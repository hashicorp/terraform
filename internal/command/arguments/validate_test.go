// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseValidate_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Validate
	}{
		"defaults": {
			nil,
			&Validate{
				Path:          ".",
				TestDirectory: "tests",
				Vars:          &Vars{},
				ViewType:      ViewHuman,
			},
		},
		"json": {
			[]string{"-json"},
			&Validate{
				Path:          ".",
				TestDirectory: "tests",
				Vars:          &Vars{},
				ViewType:      ViewJSON,
			},
		},
		"path": {
			[]string{"-json", "foo"},
			&Validate{
				Path:          "foo",
				TestDirectory: "tests",
				Vars:          &Vars{},
				ViewType:      ViewJSON,
			},
		},
		"test-directory": {
			[]string{"-test-directory", "other"},
			&Validate{
				Path:          ".",
				TestDirectory: "other",
				Vars:          &Vars{},
				ViewType:      ViewHuman,
			},
		},
		"no-tests": {
			[]string{"-no-tests"},
			&Validate{
				Path:          ".",
				TestDirectory: "tests",
				Vars:          &Vars{},
				ViewType:      ViewHuman,
				NoTests:       true,
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseValidate(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseValidate_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Validate
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop"},
			&Validate{
				Path:          ".",
				TestDirectory: "tests",
				Vars:          &Vars{},
				ViewType:      ViewHuman,
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
			[]string{"-json", "bar", "baz"},
			&Validate{
				Path:          "bar",
				TestDirectory: "tests",
				Vars:          &Vars{},
				ViewType:      ViewJSON,
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

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseValidate(tc.args)
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

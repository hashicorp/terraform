// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseShow_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Show
	}{
		"defaults": {
			nil,
			&Show{
				Path:     "",
				ViewType: ViewHuman,
				Vars:     &Vars{},
			},
		},
		"json": {
			[]string{"-json"},
			&Show{
				Path:            "",
				ViewType:        ViewJSON,
				RedactSensitive: false,
				Vars:            &Vars{},
			},
		},
		"json redacted": {
			[]string{"-json-redacted"},
			&Show{
				Path:            "",
				ViewType:        ViewJSON,
				RedactSensitive: true,
				Vars:            &Vars{},
			},
		},
		"path": {
			[]string{"-json", "foo"},
			&Show{
				Path:            "foo",
				ViewType:        ViewJSON,
				RedactSensitive: false,
				Vars:            &Vars{},
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Operation{}, Vars{}, State{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseShow(tc.args)
			if len(diags) > 0 && diags.HasErrors() {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Errorf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseShow_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Show
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop"},
			&Show{
				Path:            "",
				ViewType:        ViewHuman,
				RedactSensitive: false,
				Vars:            &Vars{},
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
			&Show{
				Path:            "bar",
				ViewType:        ViewJSON,
				RedactSensitive: false,
				Vars:            &Vars{},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Too many command line arguments",
					"Expected at most one positional argument.",
				),
			},
		},
		"incompatible flags": {
			[]string{"-json", "-json-redacted"},
			&Show{
				Path:            "",
				ViewType:        ViewJSON,
				RedactSensitive: true,
				Vars:            &Vars{},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Incompatible command-line flags",
					"The -json and -json-redacted options cannot be used together.",
				),
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Operation{}, Vars{}, State{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseShow(tc.args)
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Errorf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

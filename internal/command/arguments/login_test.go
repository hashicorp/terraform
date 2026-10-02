// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseLogin_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Login
	}{
		"default host": {
			nil,
			&Login{
				Host:         "app.terraform.io",
				InputEnabled: true,
			},
		},
		"non-default host": {
			[]string{"other.host.io"},
			&Login{
				Host:         "other.host.io",
				InputEnabled: true,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseLogin(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseLogin_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Login
		wantDiags tfdiags.Diagnostics
	}{
		"invalid flag": {
			[]string{"-foobar"},
			&Login{
				Host:         "app.terraform.io",
				InputEnabled: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -foobar",
				),
			},
		},
		"too many arguments": {
			[]string{"other.host.io", "app.terraform.io"},
			&Login{
				Host:         "other.host.io",
				InputEnabled: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"The login command expects at most one argument: the host to log in to.",
					"",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseLogin(tc.args)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseLogout_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Logout
	}{
		"default host": {
			nil,
			&Logout{
				Host: "app.terraform.io",
			},
		},
		"non-default host": {
			[]string{"other.host.io"},
			&Logout{
				Host: "other.host.io",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseLogout(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseLogout_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Logout
		wantDiags tfdiags.Diagnostics
	}{
		"invalid flag": {
			[]string{"-foobar"},
			&Logout{
				Host: "app.terraform.io",
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
			&Logout{
				Host: "other.host.io",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"The logout command expects at most one argument: the host to log out of.",
					"",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseLogout(tc.args)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

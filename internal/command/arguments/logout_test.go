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
		"defaults": {
			nil,
			&Logout{
				Hostname: "app.terraform.io",
			},
		},
		"custom hostname": {
			[]string{"registry.example.com"},
			&Logout{
				Hostname: "registry.example.com",
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
		"too many args": {
			[]string{"host1.example.com", "host2.example.com"},
			&Logout{},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid arguments",
					"The logout command expects at most one argument: the host to log out of.",
				),
			},
		},
		"unknown flag": {
			[]string{"-wat"},
			&Logout{
				Hostname: "app.terraform.io",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -wat",
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

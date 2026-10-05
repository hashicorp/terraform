// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseStateIdentities_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *StateIdentities
	}{
		"default (with -json)": {
			[]string{"-json"},
			&StateIdentities{
				ViewType: ViewJSON,
			},
		},
		"state path": {
			[]string{"-json", "-state=foobar.tfstate"},
			&StateIdentities{
				ViewType:  ViewJSON,
				StatePath: "foobar.tfstate",
			},
		},
		"id filter": {
			[]string{"-json", "-id=bar"},
			&StateIdentities{
				ViewType: ViewJSON,
				ID:       "bar",
			},
		},
		"with addresses": {
			[]string{"-json", "module.example", "aws_instance.foo"},
			&StateIdentities{
				ViewType: ViewJSON,
				Addrs:    []string{"module.example", "aws_instance.foo"},
			},
		},
		"all options": {
			[]string{"-json", "-state=foobar.tfstate", "-id=bar", "module.example"},
			&StateIdentities{
				ViewType:  ViewJSON,
				StatePath: "foobar.tfstate",
				ID:        "bar",
				Addrs:     []string{"module.example"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseStateIdentities(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseStateIdentities_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *StateIdentities
		wantDiags tfdiags.Diagnostics
	}{
		"missing -json flag": {
			nil,
			&StateIdentities{
				ViewType: ViewHuman,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Missing required -json flag",
					"The `terraform state identities` command requires the `-json` flag.",
				),
			},
		},
		"unknown flag": {
			[]string{"-json", "-boop"},
			&StateIdentities{
				ViewType: ViewJSON,
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

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseStateIdentities(tc.args)
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

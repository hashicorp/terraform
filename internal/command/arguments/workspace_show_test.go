// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseWorkspaceShow_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *WorkspaceShow
	}{
		"defaults": {
			nil,
			&WorkspaceShow{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
			},
		},
		"currently there is no validation about too many arguments": {
			[]string{"bar"},
			&WorkspaceShow{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseWorkspaceShow(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseWorkspaceShow_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *WorkspaceShow
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop"},
			&WorkspaceShow{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
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
			got, gotDiags := ParseWorkspaceShow(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

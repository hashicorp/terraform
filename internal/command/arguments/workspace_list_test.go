// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseWorkspaceList_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *WorkspaceList
	}{
		"defaults": {
			nil,
			&WorkspaceList{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
			},
		},
		"json": {
			[]string{"-json"},
			&WorkspaceList{
				Workspace: Workspace{
					ViewType: ViewJSON,
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseWorkspaceList(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseWorkspaceList_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *WorkspaceList
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop"},
			&WorkspaceList{
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
		"too many arguments": {
			[]string{"-json", "bar", "baz"},
			&WorkspaceList{
				Workspace: Workspace{
					ViewType: ViewJSON, // -json flag parsed correctly
				},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Too many command line arguments. Did you mean to use -chdir?",
					"", // No detail
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseWorkspaceList(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

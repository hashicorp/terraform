// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseWorkspaceSelect_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *WorkspaceSelect
	}{
		"name specified & default flags": {
			[]string{"my-new-workspace"},
			&WorkspaceSelect{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name:     "my-new-workspace",
				OrCreate: false,
			},
		},
		"or-create flag specified": {
			[]string{"-or-create", "my-new-workspace"},
			&WorkspaceSelect{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name:     "my-new-workspace",
				OrCreate: true,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseWorkspaceSelect(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseWorkspaceSelect_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *WorkspaceSelect
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop", "my-new-workspace"},
			&WorkspaceSelect{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name: "my-new-workspace",
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
			[]string{"my-new-workspace", "bar"},
			&WorkspaceSelect{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name: "my-new-workspace",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Expected a single argument: NAME.",
					"", // No detail
				),
			},
		},
		"missing argument": {
			[]string{},
			&WorkspaceSelect{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name: "",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Expected a single argument: NAME.",
					"", // No detail
				),
			},
		},
		"invalid workspace name": {
			[]string{""}, // empty string
			&WorkspaceSelect{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"\nThe workspace name \"\" is not allowed. The name must contain only URL safe\ncharacters, contain no path separators, and not be an empty string.\n",
					"", // No detail
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseWorkspaceSelect(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

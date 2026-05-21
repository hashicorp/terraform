// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseWorkspaceDelete_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *WorkspaceDelete
	}{
		"name specified & default flags": {
			[]string{"my-new-workspace"},
			&WorkspaceDelete{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name:        "my-new-workspace",
				Lock:        true,
				Force:       false,
				LockTimeout: 0,
			},
		},
		"invalid names are tolerated during delete": {
			[]string{"§@!invalid-name!@§"},
			&WorkspaceDelete{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name:        "§@!invalid-name!@§",
				Lock:        true,
				Force:       false,
				LockTimeout: 0,
			},
		},
		"lock flag specified": {
			[]string{"-lock=false", "my-new-workspace"},
			&WorkspaceDelete{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name:        "my-new-workspace",
				Lock:        false,
				Force:       false,
				LockTimeout: 0,
			},
		},
		"force flag specified": {
			[]string{"-force=true", "my-new-workspace"},
			&WorkspaceDelete{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name:        "my-new-workspace",
				Lock:        true,
				Force:       true,
				LockTimeout: 0,
			},
		},
		"lock-timeout flag specified": {
			[]string{"-lock-timeout=30s", "my-new-workspace"},
			&WorkspaceDelete{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name:        "my-new-workspace",
				Lock:        true,
				Force:       false,
				LockTimeout: 30 * time.Second,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseWorkspaceDelete(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseWorkspaceDelete_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *WorkspaceDelete
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop", "my-new-workspace"},
			&WorkspaceDelete{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name:        "my-new-workspace",
				Force:       false,
				Lock:        true,
				LockTimeout: 0,
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
			&WorkspaceDelete{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name:        "my-new-workspace", // First positional argument is still captured``
				Force:       false,
				Lock:        true,
				LockTimeout: 0,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Expected a single argument: NAME.",
					"", // No detail
				),
			},
		},
		"no arguments": {
			[]string{},
			&WorkspaceDelete{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Force:       false,
				Lock:        true,
				LockTimeout: 0,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Expected a single argument: NAME.",
					"", // No detail
				),
			},
		},
		"empty string as workspace name": {
			[]string{""}, // empty string
			&WorkspaceDelete{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Force:       false,
				Lock:        true,
				LockTimeout: 0,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Expected a workspace name as an argument, instead got an empty string: \"\"\n",
					"", // No detail
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseWorkspaceDelete(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

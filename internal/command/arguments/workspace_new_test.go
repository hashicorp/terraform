// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseWorkspaceNew_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *WorkspaceNew
	}{
		"name specified & default flags": {
			[]string{"my-new-workspace"},
			&WorkspaceNew{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name:        "my-new-workspace",
				Lock:        true,
				LockTimeout: 0,
				StatePath:   "",
			},
		},
		"locking turned off via -lock": {
			[]string{"-lock=false", "my-new-workspace"},
			&WorkspaceNew{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name:        "my-new-workspace",
				Lock:        false,
				LockTimeout: 0,
				StatePath:   "",
			},
		},
		"lock timeout specified via -lock-timeout": {
			[]string{"-lock-timeout=30s", "my-new-workspace"},
			&WorkspaceNew{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name:        "my-new-workspace",
				Lock:        true,
				LockTimeout: 30 * time.Second,
				StatePath:   "",
			},
		},
		"state path specified via -state": {
			[]string{"-state=path/to/state", "my-new-workspace"},
			&WorkspaceNew{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name:        "my-new-workspace",
				Lock:        true,
				LockTimeout: 0,
				StatePath:   "path/to/state",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseWorkspaceNew(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseWorkspaceNew_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *WorkspaceNew
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop", "my-new-workspace"},
			&WorkspaceNew{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name: "my-new-workspace",
				Lock: true,
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
			&WorkspaceNew{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name: "", // Isn't set if there are extra arguments supplied
				Lock: true,
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
			[]string{"§@!invalid-name!@§"},
			&WorkspaceNew{
				Workspace: Workspace{
					ViewType: ViewHuman,
				},
				Name: "§@!invalid-name!@§",
				Lock: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"\nThe workspace name \"§@!invalid-name!@§\" is not allowed. The name must contain only URL safe\ncharacters, contain no path separators, and not be an empty string.\n",
					"", // No detail
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseWorkspaceNew(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

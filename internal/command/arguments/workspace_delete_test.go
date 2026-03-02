// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseWorkspaceDelete_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *WorkspaceDelete
	}{
		"name only": {
			[]string{"myworkspace"},
			&WorkspaceDelete{
				Name:      "myworkspace",
				StateLock: true,
				Args:      []string{},
			},
		},
		"name with force": {
			[]string{"-force", "myworkspace"},
			&WorkspaceDelete{
				Name:      "myworkspace",
				Force:     true,
				StateLock: true,
				Args:      []string{},
			},
		},
		"name with lock options": {
			[]string{"-lock=false", "-lock-timeout=10s", "myworkspace"},
			&WorkspaceDelete{
				Name:             "myworkspace",
				StateLock:        false,
				StateLockTimeout: 10 * time.Second,
				Args:             []string{},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseWorkspaceDelete(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
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
		"no args": {
			[]string{},
			&WorkspaceDelete{
				StateLock: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid arguments",
					"Expected a single argument: NAME.",
				),
			},
		},
		"too many args": {
			[]string{"one", "two"},
			&WorkspaceDelete{
				StateLock: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid arguments",
					"Expected a single argument: NAME.",
				),
			},
		},
		"unknown flag": {
			[]string{"-wat"},
			&WorkspaceDelete{
				StateLock: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -wat",
				),
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid arguments",
					"Expected a single argument: NAME.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseWorkspaceDelete(tc.args)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

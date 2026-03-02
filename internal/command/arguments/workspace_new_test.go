// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseWorkspaceNew_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *WorkspaceNew
	}{
		"name only": {
			[]string{"myworkspace"},
			&WorkspaceNew{
				StateLock: true,
				Name:      "myworkspace",
				Args:      []string{},
			},
		},
		"all flags": {
			[]string{
				"-lock=false",
				"-lock-timeout=10s",
				"-state=terraform.tfstate",
				"myworkspace",
			},
			&WorkspaceNew{
				StateLock:        false,
				StateLockTimeout: 10 * time.Second,
				StatePath:        "terraform.tfstate",
				Name:             "myworkspace",
				Args:             []string{},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseWorkspaceNew(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
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
		"no args": {
			nil,
			&WorkspaceNew{
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
			&WorkspaceNew{
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
			&WorkspaceNew{
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
			got, gotDiags := ParseWorkspaceNew(tc.args)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

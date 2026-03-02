// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseWorkspaceSelect_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *WorkspaceSelect
	}{
		"name only": {
			[]string{"myworkspace"},
			&WorkspaceSelect{
				Name: "myworkspace",
				Args: []string{},
			},
		},
		"name with or-create": {
			[]string{"-or-create", "myworkspace"},
			&WorkspaceSelect{
				OrCreate: true,
				Name:     "myworkspace",
				Args:     []string{},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseWorkspaceSelect(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
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
		"no args": {
			nil,
			&WorkspaceSelect{},
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
			&WorkspaceSelect{},
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
			&WorkspaceSelect{},
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
			got, gotDiags := ParseWorkspaceSelect(tc.args)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

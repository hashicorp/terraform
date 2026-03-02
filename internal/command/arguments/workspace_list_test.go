// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseWorkspaceList_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *WorkspaceList
	}{
		"defaults": {
			nil,
			&WorkspaceList{},
		},
		"with args": {
			[]string{"some/path"},
			&WorkspaceList{
				Args: []string{"some/path"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseWorkspaceList(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
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
			[]string{"-wat"},
			&WorkspaceList{
				Args: []string{},
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
			got, gotDiags := ParseWorkspaceList(tc.args)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

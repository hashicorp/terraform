// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseForceUnlock_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *ForceUnlock
	}{
		"lock id only": {
			[]string{"abc123"},
			&ForceUnlock{
				LockID: "abc123",
				Args:   []string{},
			},
		},
		"lock id with force": {
			[]string{"-force", "abc123"},
			&ForceUnlock{
				Force:  true,
				LockID: "abc123",
				Args:   []string{},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseForceUnlock(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseForceUnlock_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *ForceUnlock
		wantDiags tfdiags.Diagnostics
	}{
		"no args": {
			nil,
			&ForceUnlock{},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid arguments",
					"Expected a single argument: LOCK_ID",
				),
			},
		},
		"too many args": {
			[]string{"abc123", "def456"},
			&ForceUnlock{},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid arguments",
					"Expected a single argument: LOCK_ID",
				),
			},
		},
		"unknown flag": {
			[]string{"-wat"},
			&ForceUnlock{},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -wat",
				),
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid arguments",
					"Expected a single argument: LOCK_ID",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseForceUnlock(tc.args)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

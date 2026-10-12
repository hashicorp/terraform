// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseUnlock_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Unlock
	}{
		"defaults with lock ID": {
			[]string{"foobar12345"},
			&Unlock{
				LockID: "foobar12345",
			},
		},
		"force unlock": {
			[]string{"-force", "foobar12345"},
			&Unlock{
				LockID: "foobar12345",
				Force:  true,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseUnlock(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseUnlock_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Unlock
		wantDiags tfdiags.Diagnostics
	}{
		"missing lock id": {
			[]string{},
			&Unlock{},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid number of arguments",
					"Expected a single argument: LOCK_ID",
				),
			},
		},
		"unknown flag": {
			[]string{"-unknown", "foobar12345"},
			&Unlock{
				LockID: "foobar12345",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -unknown",
				),
			},
		},
		"too many arguments": {
			[]string{"foobar12345", "fizzbuzz12345"},
			&Unlock{
				LockID: "",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid number of arguments",
					"Expected a single argument: LOCK_ID",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseUnlock(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

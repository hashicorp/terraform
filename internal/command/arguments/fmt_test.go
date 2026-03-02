// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseFmt_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Fmt
	}{
		"defaults": {
			nil,
			&Fmt{
				List:  true,
				Write: true,
			},
		},
		"all options": {
			[]string{
				"-list=false",
				"-write=false",
				"-diff",
				"-check",
				"-recursive",
			},
			&Fmt{
				List:      false,
				Write:     false,
				Diff:      true,
				Check:     true,
				Recursive: true,
				Paths:     []string{},
			},
		},
		"with paths": {
			[]string{
				"-diff",
				"dir1",
				"dir2",
			},
			&Fmt{
				List:  true,
				Write: true,
				Diff:  true,
				Paths: []string{"dir1", "dir2"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseFmt(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseFmt_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Fmt
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-wat"},
			&Fmt{
				List:  true,
				Write: true,
				Paths: []string{},
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
			got, gotDiags := ParseFmt(tc.args)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

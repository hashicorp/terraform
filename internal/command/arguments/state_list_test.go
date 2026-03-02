// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseStateList_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *StateList
	}{
		"defaults": {
			nil,
			&StateList{},
		},
		"state path": {
			[]string{"-state=foobar.tfstate"},
			&StateList{
				StatePath: "foobar.tfstate",
			},
		},
		"id filter": {
			[]string{"-id=bar"},
			&StateList{
				ID: "bar",
			},
		},
		"with addresses": {
			[]string{"module.example", "aws_instance.foo"},
			&StateList{
				Addrs: []string{"module.example", "aws_instance.foo"},
			},
		},
		"all options": {
			[]string{"-state=foobar.tfstate", "-id=bar", "module.example"},
			&StateList{
				StatePath: "foobar.tfstate",
				ID:        "bar",
				Addrs:     []string{"module.example"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseStateList(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if got.StatePath != tc.want.StatePath {
				t.Fatalf("unexpected StatePath\n got: %q\nwant: %q", got.StatePath, tc.want.StatePath)
			}
			if got.ID != tc.want.ID {
				t.Fatalf("unexpected ID\n got: %q\nwant: %q", got.ID, tc.want.ID)
			}
			if len(got.Addrs) != len(tc.want.Addrs) {
				t.Fatalf("unexpected Addrs length\n got: %d\nwant: %d", len(got.Addrs), len(tc.want.Addrs))
			}
			for i := range got.Addrs {
				if got.Addrs[i] != tc.want.Addrs[i] {
					t.Fatalf("unexpected Addrs[%d]\n got: %q\nwant: %q", i, got.Addrs[i], tc.want.Addrs[i])
				}
			}
		})
	}
}

func TestParseStateList_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *StateList
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop"},
			&StateList{},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -boop",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseStateList(tc.args)
			if got.StatePath != tc.want.StatePath {
				t.Fatalf("unexpected StatePath\n got: %q\nwant: %q", got.StatePath, tc.want.StatePath)
			}
			if got.ID != tc.want.ID {
				t.Fatalf("unexpected ID\n got: %q\nwant: %q", got.ID, tc.want.ID)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

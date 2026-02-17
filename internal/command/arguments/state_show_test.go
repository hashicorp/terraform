// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseStateShow_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *StateShow
	}{
		"address only": {
			[]string{"test_instance.foo"},
			&StateShow{
				Address: "test_instance.foo",
			},
		},
		"with state path": {
			[]string{"-state=foobar.tfstate", "test_instance.foo"},
			&StateShow{
				StatePath: "foobar.tfstate",
				Address:   "test_instance.foo",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseStateShow(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseStateShow_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *StateShow
		wantDiags tfdiags.Diagnostics
	}{
		"no arguments": {
			nil,
			&StateShow{},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"Exactly one argument expected: the address of a resource instance to show.",
				),
			},
		},
		"too many arguments": {
			[]string{"test_instance.foo", "test_instance.bar"},
			&StateShow{
				Address: "test_instance.foo",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"Exactly one argument expected: the address of a resource instance to show.",
				),
			},
		},
		"unknown flag": {
			[]string{"-boop"},
			&StateShow{},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -boop",
				),
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"Exactly one argument expected: the address of a resource instance to show.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseStateShow(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

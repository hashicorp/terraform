// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseStatePush_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *StatePush
	}{
		"file path": {
			[]string{"state.tfstate"},
			&StatePush{
				Force:     false,
				StateLock: true,
				Path:      "state.tfstate",
			},
		},
		"stdin": {
			[]string{"-"},
			&StatePush{
				StateLock: true,
				Path:      "-",
			},
		},
		"all flags": {
			[]string{
				"-force",
				"-lock=false",
				"-lock-timeout=10s",
				"-ignore-remote-version",
				"state.tfstate",
			},
			&StatePush{
				Force:               true,
				StateLock:           false,
				StateLockTimeout:    10_000_000_000,
				IgnoreRemoteVersion: true,
				Path:                "state.tfstate",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseStatePush(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseStatePush_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *StatePush
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-unknown", "state.tfstate"},
			&StatePush{
				StateLock: true,
				Path:      "state.tfstate",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -unknown",
				),
			},
		},
		"no arguments": {
			nil,
			&StatePush{
				StateLock: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Exactly one argument expected",
					"The state push command requires a path to a local state file to push. Use \"-\" to read from stdin.",
				),
			},
		},
		"too many arguments": {
			[]string{"a", "b"},
			&StatePush{
				StateLock: true,
				Path:      "a",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Exactly one argument expected",
					"The state push command requires a path to a local state file to push. Use \"-\" to read from stdin.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseStatePush(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

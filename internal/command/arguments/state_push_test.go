// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseStatePush_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *StatePush
	}{
		"path only": {
			[]string{"replace.tfstate"},
			&StatePush{
				StateLock: true,
				Path:      "replace.tfstate",
			},
		},
		"stdin": {
			[]string{"-"},
			&StatePush{
				StateLock: true,
				Path:      "-",
			},
		},
		"force": {
			[]string{"-force", "replace.tfstate"},
			&StatePush{
				Force:     true,
				StateLock: true,
				Path:      "replace.tfstate",
			},
		},
		"lock disabled": {
			[]string{"-lock=false", "replace.tfstate"},
			&StatePush{
				Path: "replace.tfstate",
			},
		},
		"lock timeout": {
			[]string{"-lock-timeout=5s", "replace.tfstate"},
			&StatePush{
				StateLock:        true,
				StateLockTimeout: 5 * time.Second,
				Path:             "replace.tfstate",
			},
		},
		"ignore remote version": {
			[]string{"-ignore-remote-version", "replace.tfstate"},
			&StatePush{
				StateLock:           true,
				IgnoreRemoteVersion: true,
				Path:                "replace.tfstate",
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
		"no arguments": {
			nil,
			&StatePush{
				StateLock: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"Exactly one argument expected: the path to a Terraform state file.",
				),
			},
		},
		"too many arguments": {
			[]string{"foo.tfstate", "bar.tfstate"},
			&StatePush{
				StateLock: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"Exactly one argument expected: the path to a Terraform state file.",
				),
			},
		},
		"unknown flag": {
			[]string{"-boop"},
			&StatePush{
				StateLock: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -boop",
				),
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"Exactly one argument expected: the path to a Terraform state file.",
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

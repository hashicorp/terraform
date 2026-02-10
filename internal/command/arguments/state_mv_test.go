// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseStateMv_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *StateMv
	}{
		"defaults with positionals": {
			[]string{"source.addr", "dest.addr"},
			&StateMv{
				DryRun:        false,
				BackupPath:    "-",
				BackupPathOut: "-",
				StateLock:     true,
				StatePath:     "",
				StatePathOut:  "",
				Source:        "source.addr",
				Destination:   "dest.addr",
			},
		},
		"dry-run": {
			[]string{"-dry-run", "source.addr", "dest.addr"},
			&StateMv{
				DryRun:        true,
				BackupPath:    "-",
				BackupPathOut: "-",
				StateLock:     true,
				Source:        "source.addr",
				Destination:   "dest.addr",
			},
		},
		"all flags": {
			[]string{
				"-dry-run",
				"-backup=backup.tfstate",
				"-backup-out=backup-out.tfstate",
				"-lock=false",
				"-lock-timeout=10s",
				"-state=state.tfstate",
				"-state-out=state-out.tfstate",
				"-ignore-remote-version",
				"source.addr",
				"dest.addr",
			},
			&StateMv{
				DryRun:              true,
				BackupPath:          "backup.tfstate",
				BackupPathOut:       "backup-out.tfstate",
				StateLock:           false,
				StateLockTimeout:    10_000_000_000,
				StatePath:           "state.tfstate",
				StatePathOut:        "state-out.tfstate",
				IgnoreRemoteVersion: true,
				Source:              "source.addr",
				Destination:         "dest.addr",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseStateMv(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseStateMv_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *StateMv
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-unknown", "source", "dest"},
			&StateMv{
				BackupPath:    "-",
				BackupPathOut: "-",
				StateLock:     true,
				Source:        "source",
				Destination:   "dest",
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
			&StateMv{
				BackupPath:    "-",
				BackupPathOut: "-",
				StateLock:     true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Exactly two arguments expected",
					"The state mv command requires a source and destination address.",
				),
			},
		},
		"too few arguments": {
			[]string{"source"},
			&StateMv{
				BackupPath:    "-",
				BackupPathOut: "-",
				StateLock:     true,
				Source:        "source",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Exactly two arguments expected",
					"The state mv command requires a source and destination address.",
				),
			},
		},
		"too many arguments": {
			[]string{"a", "b", "c"},
			&StateMv{
				BackupPath:    "-",
				BackupPathOut: "-",
				StateLock:     true,
				Source:        "a",
				Destination:   "b",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Exactly two arguments expected",
					"The state mv command requires a source and destination address.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseStateMv(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

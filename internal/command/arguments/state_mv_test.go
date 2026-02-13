// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseStateMv_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *StateMv
	}{
		"addresses only": {
			[]string{"test_instance.foo", "test_instance.bar"},
			&StateMv{
				DryRun:              false,
				BackupPath:          "-",
				BackupOutPath:       "-",
				StateLock:           true,
				StateLockTimeout:    0,
				StatePath:           "",
				StateOutPath:        "",
				IgnoreRemoteVersion: false,
				SourceAddr:          "test_instance.foo",
				DestAddr:            "test_instance.bar",
			},
		},
		"dry run": {
			[]string{"-dry-run", "test_instance.foo", "test_instance.bar"},
			&StateMv{
				DryRun:              true,
				BackupPath:          "-",
				BackupOutPath:       "-",
				StateLock:           true,
				StateLockTimeout:    0,
				StatePath:           "",
				StateOutPath:        "",
				IgnoreRemoteVersion: false,
				SourceAddr:          "test_instance.foo",
				DestAddr:            "test_instance.bar",
			},
		},
		"all options": {
			[]string{
				"-dry-run",
				"-backup=backup.tfstate",
				"-backup-out=backup-out.tfstate",
				"-lock=false",
				"-lock-timeout=5s",
				"-state=state.tfstate",
				"-state-out=state-out.tfstate",
				"-ignore-remote-version",
				"test_instance.foo",
				"test_instance.bar",
			},
			&StateMv{
				DryRun:              true,
				BackupPath:          "backup.tfstate",
				BackupOutPath:       "backup-out.tfstate",
				StateLock:           false,
				StateLockTimeout:    5 * time.Second,
				StatePath:           "state.tfstate",
				StateOutPath:        "state-out.tfstate",
				IgnoreRemoteVersion: true,
				SourceAddr:          "test_instance.foo",
				DestAddr:            "test_instance.bar",
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
		"no arguments": {
			nil,
			&StateMv{
				BackupPath:    "-",
				BackupOutPath: "-",
				StateLock:     true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"Exactly two arguments expected: the source and destination addresses.",
				),
			},
		},
		"one argument": {
			[]string{"test_instance.foo"},
			&StateMv{
				BackupPath:    "-",
				BackupOutPath: "-",
				StateLock:     true,
				SourceAddr:    "test_instance.foo",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"Exactly two arguments expected: the source and destination addresses.",
				),
			},
		},
		"too many arguments": {
			[]string{"a", "b", "c"},
			&StateMv{
				BackupPath:    "-",
				BackupOutPath: "-",
				StateLock:     true,
				SourceAddr:    "a",
				DestAddr:      "b",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"Exactly two arguments expected: the source and destination addresses.",
				),
			},
		},
		"unknown flag": {
			[]string{"-boop"},
			&StateMv{
				BackupPath:    "-",
				BackupOutPath: "-",
				StateLock:     true,
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
					"Exactly two arguments expected: the source and destination addresses.",
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

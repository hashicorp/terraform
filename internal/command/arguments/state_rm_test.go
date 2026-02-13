// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseStateRm_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *StateRm
	}{
		"single address": {
			[]string{"test_instance.foo"},
			&StateRm{
				DryRun:              false,
				BackupPath:          "-",
				StateLock:           true,
				StateLockTimeout:    0,
				StatePath:           "",
				IgnoreRemoteVersion: false,
				Addrs:               []string{"test_instance.foo"},
			},
		},
		"multiple addresses": {
			[]string{"test_instance.foo", "test_instance.bar"},
			&StateRm{
				DryRun:              false,
				BackupPath:          "-",
				StateLock:           true,
				StateLockTimeout:    0,
				StatePath:           "",
				IgnoreRemoteVersion: false,
				Addrs:               []string{"test_instance.foo", "test_instance.bar"},
			},
		},
		"all options": {
			[]string{"-dry-run", "-backup=backup.tfstate", "-lock=false", "-lock-timeout=5s", "-state=state.tfstate", "-ignore-remote-version", "test_instance.foo"},
			&StateRm{
				DryRun:              true,
				BackupPath:          "backup.tfstate",
				StateLock:           false,
				StateLockTimeout:    5 * time.Second,
				StatePath:           "state.tfstate",
				IgnoreRemoteVersion: true,
				Addrs:               []string{"test_instance.foo"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseStateRm(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if got.DryRun != tc.want.DryRun ||
				got.BackupPath != tc.want.BackupPath ||
				got.StateLock != tc.want.StateLock ||
				got.StateLockTimeout != tc.want.StateLockTimeout ||
				got.StatePath != tc.want.StatePath ||
				got.IgnoreRemoteVersion != tc.want.IgnoreRemoteVersion {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
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

func TestParseStateRm_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		wantAddrs int
		wantDiags tfdiags.Diagnostics
	}{
		"no arguments": {
			nil,
			0,
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"At least one address is required.",
				),
			},
		},
		"unknown flag": {
			[]string{"-boop"},
			0,
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -boop",
				),
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"At least one address is required.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseStateRm(tc.args)
			if len(got.Addrs) != tc.wantAddrs {
				t.Fatalf("unexpected Addrs length\n got: %d\nwant: %d", len(got.Addrs), tc.wantAddrs)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

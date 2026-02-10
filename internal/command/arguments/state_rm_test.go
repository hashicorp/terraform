// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"reflect"
	"testing"

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
				DryRun:     false,
				BackupPath: "-",
				StateLock:  true,
				StatePath:  "",
				Addrs:      []string{"test_instance.foo"},
			},
		},
		"multiple addresses": {
			[]string{"test_instance.foo", "test_instance.bar"},
			&StateRm{
				BackupPath: "-",
				StateLock:  true,
				Addrs:      []string{"test_instance.foo", "test_instance.bar"},
			},
		},
		"dry-run": {
			[]string{"-dry-run", "test_instance.foo"},
			&StateRm{
				DryRun:     true,
				BackupPath: "-",
				StateLock:  true,
				Addrs:      []string{"test_instance.foo"},
			},
		},
		"all flags": {
			[]string{
				"-dry-run",
				"-backup=backup.tfstate",
				"-lock=false",
				"-lock-timeout=10s",
				"-state=state.tfstate",
				"-ignore-remote-version",
				"test_instance.foo",
			},
			&StateRm{
				DryRun:              true,
				BackupPath:          "backup.tfstate",
				StateLock:           false,
				StateLockTimeout:    10_000_000_000,
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
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseStateRm_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *StateRm
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-unknown", "test_instance.foo"},
			&StateRm{
				BackupPath: "-",
				StateLock:  true,
				Addrs:      []string{"test_instance.foo"},
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
			&StateRm{
				BackupPath: "-",
				StateLock:  true,
				Addrs:      nil,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"At least one address required",
					"The state rm command requires one or more resource addresses as arguments.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseStateRm(tc.args)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

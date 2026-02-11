// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseUntaint_valid(t *testing.T) {
	testCases := map[string]struct {
		args             []string
		wantAddr         string
		wantAllowMissing bool
		wantStatePath    string
		wantStateOut     string
		wantBackup       string
		wantLock         bool
		wantLockTimeout  time.Duration
	}{
		"resource address only": {
			args:     []string{"test_instance.foo"},
			wantAddr: "test_instance.foo",
			wantLock: true,
		},
		"allow-missing": {
			args:             []string{"-allow-missing", "test_instance.foo"},
			wantAddr:         "test_instance.foo",
			wantAllowMissing: true,
			wantLock:         true,
		},
		"state path": {
			args:          []string{"-state", "custom.tfstate", "test_instance.foo"},
			wantAddr:      "test_instance.foo",
			wantStatePath: "custom.tfstate",
			wantLock:      true,
		},
		"state-out path": {
			args:         []string{"-state-out", "out.tfstate", "test_instance.foo"},
			wantAddr:     "test_instance.foo",
			wantStateOut: "out.tfstate",
			wantLock:     true,
		},
		"backup path": {
			args:       []string{"-backup", "backup.tfstate", "test_instance.foo"},
			wantAddr:   "test_instance.foo",
			wantBackup: "backup.tfstate",
			wantLock:   true,
		},
		"disable backup": {
			args:       []string{"-backup", "-", "test_instance.foo"},
			wantAddr:   "test_instance.foo",
			wantBackup: "-",
			wantLock:   true,
		},
		"lock disabled": {
			args:     []string{"-lock=false", "test_instance.foo"},
			wantAddr: "test_instance.foo",
			wantLock: false,
		},
		"lock-timeout": {
			args:            []string{"-lock-timeout=10s", "test_instance.foo"},
			wantAddr:        "test_instance.foo",
			wantLock:        true,
			wantLockTimeout: 10 * time.Second,
		},
		"ignore-remote-version": {
			args:     []string{"-ignore-remote-version", "test_instance.foo"},
			wantAddr: "test_instance.foo",
			wantLock: true,
		},
		"module address": {
			args:     []string{"module.child.test_instance.foo"},
			wantAddr: "module.child.test_instance.foo",
			wantLock: true,
		},
		"all flags": {
			args: []string{
				"-allow-missing",
				"-state", "custom.tfstate",
				"-state-out", "out.tfstate",
				"-backup", "backup.tfstate",
				"-lock=false",
				"-lock-timeout=5s",
				"-ignore-remote-version",
				"test_instance.foo",
			},
			wantAddr:         "test_instance.foo",
			wantAllowMissing: true,
			wantStatePath:    "custom.tfstate",
			wantStateOut:     "out.tfstate",
			wantBackup:       "backup.tfstate",
			wantLock:         false,
			wantLockTimeout:  5 * time.Second,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseUntaint(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if got.Addr.String() != tc.wantAddr {
				t.Fatalf("unexpected addr\n got: %s\nwant: %s", got.Addr.String(), tc.wantAddr)
			}
			if got.AllowMissing != tc.wantAllowMissing {
				t.Fatalf("unexpected allow-missing\n got: %v\nwant: %v", got.AllowMissing, tc.wantAllowMissing)
			}
			if got.StatePath != tc.wantStatePath {
				t.Fatalf("unexpected state path\n got: %s\nwant: %s", got.StatePath, tc.wantStatePath)
			}
			if got.StateOutPath != tc.wantStateOut {
				t.Fatalf("unexpected state-out path\n got: %s\nwant: %s", got.StateOutPath, tc.wantStateOut)
			}
			if got.BackupPath != tc.wantBackup {
				t.Fatalf("unexpected backup path\n got: %s\nwant: %s", got.BackupPath, tc.wantBackup)
			}
			if got.Lock != tc.wantLock {
				t.Fatalf("unexpected lock\n got: %v\nwant: %v", got.Lock, tc.wantLock)
			}
			if got.LockTimeout != tc.wantLockTimeout {
				t.Fatalf("unexpected lock-timeout\n got: %v\nwant: %v", got.LockTimeout, tc.wantLockTimeout)
			}
		})
	}
}

func TestParseUntaint_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			args: []string{"-unknown"},
			wantDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -unknown",
				),
				tfdiags.Sourceless(
					tfdiags.Error,
					"Missing required argument",
					"The untaint command expects exactly one argument: the address of the resource instance to untaint.",
				),
			},
		},
		"no arguments": {
			args: nil,
			wantDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Missing required argument",
					"The untaint command expects exactly one argument: the address of the resource instance to untaint.",
				),
			},
		},
		"too many arguments": {
			args: []string{"test_instance.foo", "test_instance.bar"},
			wantDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Too many command line arguments",
					"The untaint command expects exactly one argument: the address of the resource instance to untaint.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseUntaint(tc.args)
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

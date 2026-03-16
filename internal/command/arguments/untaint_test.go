// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseUntaint_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Untaint
	}{
		"defaults with address": {
			[]string{"test_instance.foo"},
			&Untaint{
				Address:   "test_instance.foo",
				StateLock: true,
			},
		},
		"allow-missing": {
			[]string{"-allow-missing", "test_instance.foo"},
			&Untaint{
				Address:      "test_instance.foo",
				AllowMissing: true,
				StateLock:    true,
			},
		},
		"backup": {
			[]string{"-backup", "backup.tfstate", "test_instance.foo"},
			&Untaint{
				Address:    "test_instance.foo",
				BackupPath: "backup.tfstate",
				StateLock:  true,
			},
		},
		"lock disabled": {
			[]string{"-lock=false", "test_instance.foo"},
			&Untaint{
				Address: "test_instance.foo",
			},
		},
		"lock-timeout": {
			[]string{"-lock-timeout=10s", "test_instance.foo"},
			&Untaint{
				Address:          "test_instance.foo",
				StateLock:        true,
				StateLockTimeout: 10 * time.Second,
			},
		},
		"state": {
			[]string{"-state=foo.tfstate", "test_instance.foo"},
			&Untaint{
				Address:   "test_instance.foo",
				StateLock: true,
				StatePath: "foo.tfstate",
			},
		},
		"state-out": {
			[]string{"-state-out=foo.tfstate", "test_instance.foo"},
			&Untaint{
				Address:      "test_instance.foo",
				StateLock:    true,
				StateOutPath: "foo.tfstate",
			},
		},
		"ignore-remote-version": {
			[]string{"-ignore-remote-version", "test_instance.foo"},
			&Untaint{
				Address:             "test_instance.foo",
				StateLock:           true,
				IgnoreRemoteVersion: true,
			},
		},
		"all flags": {
			[]string{
				"-allow-missing",
				"-backup=backup.tfstate",
				"-lock=false",
				"-lock-timeout=10s",
				"-state=foo.tfstate",
				"-state-out=bar.tfstate",
				"-ignore-remote-version",
				"module.child.test_instance.foo",
			},
			&Untaint{
				Address:             "module.child.test_instance.foo",
				AllowMissing:        true,
				BackupPath:          "backup.tfstate",
				StateLockTimeout:    10 * time.Second,
				StatePath:           "foo.tfstate",
				StateOutPath:        "bar.tfstate",
				IgnoreRemoteVersion: true,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseUntaint(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseUntaint_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Untaint
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-unknown"},
			&Untaint{
				StateLock: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -unknown",
				),
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"The untaint command expects exactly one argument: the address of the resource to untaint.",
				),
			},
		},
		"missing address": {
			nil,
			&Untaint{
				StateLock: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"The untaint command expects exactly one argument: the address of the resource to untaint.",
				),
			},
		},
		"too many arguments": {
			[]string{"test_instance.foo", "test_instance.bar"},
			&Untaint{
				Address:   "test_instance.foo",
				StateLock: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Too many command line arguments",
					"The untaint command expects exactly one argument: the address of the resource to untaint.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseUntaint(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

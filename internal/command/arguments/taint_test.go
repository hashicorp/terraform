// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseTaint_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Taint
	}{
		"defaults with address": {
			[]string{"test_instance.foo"},
			&Taint{
				Address:   "test_instance.foo",
				StateLock: true,
			},
		},
		"allow-missing": {
			[]string{"-allow-missing", "test_instance.foo"},
			&Taint{
				Address:      "test_instance.foo",
				AllowMissing: true,
				StateLock:    true,
			},
		},
		"backup": {
			[]string{"-backup", "backup.tfstate", "test_instance.foo"},
			&Taint{
				Address:    "test_instance.foo",
				BackupPath: "backup.tfstate",
				StateLock:  true,
			},
		},
		"lock disabled": {
			[]string{"-lock=false", "test_instance.foo"},
			&Taint{
				Address:   "test_instance.foo",
				StateLock: false,
			},
		},
		"lock-timeout": {
			[]string{"-lock-timeout=10s", "test_instance.foo"},
			&Taint{
				Address:          "test_instance.foo",
				StateLock:        true,
				StateLockTimeout: 10 * time.Second,
			},
		},
		"state": {
			[]string{"-state=foo.tfstate", "test_instance.foo"},
			&Taint{
				Address:   "test_instance.foo",
				StateLock: true,
				StatePath: "foo.tfstate",
			},
		},
		"state-out": {
			[]string{"-state-out=foo.tfstate", "test_instance.foo"},
			&Taint{
				Address:      "test_instance.foo",
				StateLock:    true,
				StateOutPath: "foo.tfstate",
			},
		},
		"ignore-remote-version": {
			[]string{"-ignore-remote-version", "test_instance.foo"},
			&Taint{
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
			&Taint{
				Address:             "module.child.test_instance.foo",
				AllowMissing:        true,
				BackupPath:          "backup.tfstate",
				StateLock:           false,
				StateLockTimeout:    10 * time.Second,
				StatePath:           "foo.tfstate",
				StateOutPath:        "bar.tfstate",
				IgnoreRemoteVersion: true,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseTaint(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseTaint_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Taint
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-unknown"},
			&Taint{
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
					"The taint command expects exactly one argument: the address of the resource to taint.",
				),
			},
		},
		"missing address": {
			nil,
			&Taint{
				StateLock: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"The taint command expects exactly one argument: the address of the resource to taint.",
				),
			},
		},
		"too many arguments": {
			[]string{"test_instance.foo", "test_instance.bar"},
			&Taint{
				Address:   "test_instance.foo",
				StateLock: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Too many command line arguments",
					"The taint command expects exactly one argument: the address of the resource to taint.",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseTaint(tc.args)
			if *got != *tc.want {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

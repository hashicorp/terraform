// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

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
				Vars:      &Vars{},
				Address:   "test_instance.foo",
				StateLock: true,
			},
		},
		"allow-missing": {
			[]string{"-allow-missing", "test_instance.foo"},
			&Taint{
				Vars:         &Vars{},
				Address:      "test_instance.foo",
				AllowMissing: true,
				StateLock:    true,
			},
		},
		"backup": {
			[]string{"-backup", "backup.tfstate", "test_instance.foo"},
			&Taint{
				Vars:       &Vars{},
				Address:    "test_instance.foo",
				BackupPath: "backup.tfstate",
				StateLock:  true,
			},
		},
		"lock disabled": {
			[]string{"-lock=false", "test_instance.foo"},
			&Taint{
				Vars:    &Vars{},
				Address: "test_instance.foo",
			},
		},
		"lock-timeout": {
			[]string{"-lock-timeout=10s", "test_instance.foo"},
			&Taint{
				Vars:             &Vars{},
				Address:          "test_instance.foo",
				StateLock:        true,
				StateLockTimeout: 10 * time.Second,
			},
		},
		"state": {
			[]string{"-state=foo.tfstate", "test_instance.foo"},
			&Taint{
				Vars:      &Vars{},
				Address:   "test_instance.foo",
				StateLock: true,
				StatePath: "foo.tfstate",
			},
		},
		"state-out": {
			[]string{"-state-out=foo.tfstate", "test_instance.foo"},
			&Taint{
				Vars:         &Vars{},
				Address:      "test_instance.foo",
				StateLock:    true,
				StateOutPath: "foo.tfstate",
			},
		},
		"ignore-remote-version": {
			[]string{"-ignore-remote-version", "test_instance.foo"},
			&Taint{
				Vars:                &Vars{},
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
				Vars:                &Vars{},
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

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseTaint(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseTaint_vars(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want []FlagNameValue
	}{
		"var": {
			args: []string{"-var", "foo=bar", "test_instance.foo"},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
			},
		},
		"var-file": {
			args: []string{"-var-file", "cool.tfvars", "test_instance.foo"},
			want: []FlagNameValue{
				{Name: "-var-file", Value: "cool.tfvars"},
			},
		},
		"both": {
			args: []string{
				"-var", "foo=bar",
				"-var-file", "cool.tfvars",
				"-var", "boop=beep",
				"test_instance.foo",
			},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
				{Name: "-var-file", Value: "cool.tfvars"},
				{Name: "-var", Value: "boop=beep"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseTaint(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if vars := got.Vars.All(); !cmp.Equal(vars, tc.want) {
				t.Fatalf("unexpected vars: %#v", vars)
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
				Vars:      &Vars{},
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
				Vars:      &Vars{},
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
				Vars:      &Vars{},
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

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseTaint(tc.args)
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

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

func TestParseStateMv_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *StateMv
	}{
		"addresses only": {
			[]string{"test_instance.foo", "test_instance.bar"},
			&StateMv{
				Vars:          &Vars{},
				BackupPath:    "-",
				BackupOutPath: "-",
				StateLock:     true,
				SourceAddr:    "test_instance.foo",
				DestAddr:      "test_instance.bar",
			},
		},
		"dry run": {
			[]string{"-dry-run", "test_instance.foo", "test_instance.bar"},
			&StateMv{
				Vars:          &Vars{},
				DryRun:        true,
				BackupPath:    "-",
				BackupOutPath: "-",
				StateLock:     true,
				SourceAddr:    "test_instance.foo",
				DestAddr:      "test_instance.bar",
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
				Vars:                &Vars{},
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

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseStateMv(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseStateMv_vars(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want []FlagNameValue
	}{
		"var": {
			args: []string{"-var", "foo=bar", "test_instance.foo", "test_instance.bar"},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
			},
		},
		"var-file": {
			args: []string{"-var-file", "cool.tfvars", "test_instance.foo", "test_instance.bar"},
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
				"test_instance.bar",
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
			got, diags := ParseStateMv(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if vars := got.Vars.All(); !cmp.Equal(vars, tc.want) {
				t.Fatalf("unexpected vars: %#v", vars)
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
				Vars:          &Vars{},
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
				Vars:          &Vars{},
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
				Vars:          &Vars{},
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
				Vars:          &Vars{},
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

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseStateMv(tc.args)
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

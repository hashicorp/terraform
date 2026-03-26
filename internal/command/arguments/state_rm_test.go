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

func TestParseStateRm_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *StateRm
	}{
		"single address": {
			[]string{"test_instance.foo"},
			&StateRm{
				Vars:       &Vars{},
				BackupPath: "-",
				StateLock:  true,
				Addrs:      []string{"test_instance.foo"},
			},
		},
		"multiple addresses": {
			[]string{"test_instance.foo", "test_instance.bar"},
			&StateRm{
				Vars:       &Vars{},
				BackupPath: "-",
				StateLock:  true,
				Addrs:      []string{"test_instance.foo", "test_instance.bar"},
			},
		},
		"all options": {
			[]string{"-dry-run", "-backup=backup.tfstate", "-lock=false", "-lock-timeout=5s", "-state=state.tfstate", "-ignore-remote-version", "test_instance.foo"},
			&StateRm{
				Vars:                &Vars{},
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

	cmpOpts := cmp.Options{
		cmpopts.IgnoreUnexported(Vars{}),
		cmpopts.EquateEmpty(),
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseStateRm(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseStateRm_vars(t *testing.T) {
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
			got, diags := ParseStateRm(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if vars := got.Vars.All(); !cmp.Equal(vars, tc.want) {
				t.Fatalf("unexpected vars: %#v", vars)
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
		"no arguments": {
			nil,
			&StateRm{
				Vars:       &Vars{},
				BackupPath: "-",
				StateLock:  true,
			},
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
			&StateRm{
				Vars:       &Vars{},
				BackupPath: "-",
				StateLock:  true,
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
					"At least one address is required.",
				),
			},
		},
	}

	cmpOpts := cmp.Options{
		cmpopts.IgnoreUnexported(Vars{}),
		cmpopts.EquateEmpty(),
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseStateRm(tc.args)
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

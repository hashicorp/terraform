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

func TestParseStateReplaceProvider_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *StateReplaceProvider
	}{
		"provider addresses only": {
			[]string{"hashicorp/aws", "acmecorp/aws"},
			&StateReplaceProvider{
				Vars:             &Vars{},
				BackupPath:       "-",
				StateLock:        true,
				FromProviderAddr: "hashicorp/aws",
				ToProviderAddr:   "acmecorp/aws",
			},
		},
		"auto approve": {
			[]string{"-auto-approve", "hashicorp/aws", "acmecorp/aws"},
			&StateReplaceProvider{
				Vars:             &Vars{},
				AutoApprove:      true,
				BackupPath:       "-",
				StateLock:        true,
				FromProviderAddr: "hashicorp/aws",
				ToProviderAddr:   "acmecorp/aws",
			},
		},
		"all options": {
			[]string{
				"-auto-approve",
				"-backup=backup.tfstate",
				"-lock=false",
				"-lock-timeout=5s",
				"-state=state.tfstate",
				"-ignore-remote-version",
				"hashicorp/aws",
				"acmecorp/aws",
			},
			&StateReplaceProvider{
				Vars:                &Vars{},
				AutoApprove:         true,
				BackupPath:          "backup.tfstate",
				StateLock:           false,
				StateLockTimeout:    5 * time.Second,
				StatePath:           "state.tfstate",
				IgnoreRemoteVersion: true,
				FromProviderAddr:    "hashicorp/aws",
				ToProviderAddr:      "acmecorp/aws",
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseStateReplaceProvider(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseStateReplaceProvider_vars(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want []FlagNameValue
	}{
		"var": {
			args: []string{"-var", "foo=bar", "hashicorp/aws", "acmecorp/aws"},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
			},
		},
		"var-file": {
			args: []string{"-var-file", "cool.tfvars", "hashicorp/aws", "acmecorp/aws"},
			want: []FlagNameValue{
				{Name: "-var-file", Value: "cool.tfvars"},
			},
		},
		"both": {
			args: []string{
				"-var", "foo=bar",
				"-var-file", "cool.tfvars",
				"-var", "boop=beep",
				"hashicorp/aws", "acmecorp/aws",
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
			got, diags := ParseStateReplaceProvider(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if vars := got.Vars.All(); !cmp.Equal(vars, tc.want) {
				t.Fatalf("unexpected vars: %#v", vars)
			}
		})
	}
}

func TestParseStateReplaceProvider_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *StateReplaceProvider
		wantDiags tfdiags.Diagnostics
	}{
		"no arguments": {
			nil,
			&StateReplaceProvider{
				Vars:       &Vars{},
				BackupPath: "-",
				StateLock:  true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"Exactly two arguments expected: the from and to provider addresses.",
				),
			},
		},
		"too many arguments": {
			[]string{"a", "b", "c", "d"},
			&StateReplaceProvider{
				Vars:       &Vars{},
				BackupPath: "-",
				StateLock:  true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"Exactly two arguments expected: the from and to provider addresses.",
				),
			},
		},
		"unknown flag": {
			[]string{"-invalid", "hashicorp/google", "acmecorp/google"},
			&StateReplaceProvider{
				Vars:             &Vars{},
				BackupPath:       "-",
				StateLock:        true,
				FromProviderAddr: "hashicorp/google",
				ToProviderAddr:   "acmecorp/google",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -invalid",
				),
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseStateReplaceProvider(tc.args)
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

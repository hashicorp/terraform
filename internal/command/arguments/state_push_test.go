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

func TestParseStatePush_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *StatePush
	}{
		"path only": {
			[]string{"replace.tfstate"},
			&StatePush{
				Vars:      &Vars{},
				StateLock: true,
				Path:      "replace.tfstate",
			},
		},
		"stdin": {
			[]string{"-"},
			&StatePush{
				Vars:      &Vars{},
				StateLock: true,
				Path:      "-",
			},
		},
		"force": {
			[]string{"-force", "replace.tfstate"},
			&StatePush{
				Vars:      &Vars{},
				Force:     true,
				StateLock: true,
				Path:      "replace.tfstate",
			},
		},
		"lock disabled": {
			[]string{"-lock=false", "replace.tfstate"},
			&StatePush{
				Vars: &Vars{},
				Path: "replace.tfstate",
			},
		},
		"lock timeout": {
			[]string{"-lock-timeout=5s", "replace.tfstate"},
			&StatePush{
				Vars:             &Vars{},
				StateLock:        true,
				StateLockTimeout: 5 * time.Second,
				Path:             "replace.tfstate",
			},
		},
		"ignore remote version": {
			[]string{"-ignore-remote-version", "replace.tfstate"},
			&StatePush{
				Vars:                &Vars{},
				StateLock:           true,
				IgnoreRemoteVersion: true,
				Path:                "replace.tfstate",
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseStatePush(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseStatePush_vars(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want []FlagNameValue
	}{
		"var": {
			args: []string{"-var", "foo=bar", "replace.tfstate"},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
			},
		},
		"var-file": {
			args: []string{"-var-file", "cool.tfvars", "replace.tfstate"},
			want: []FlagNameValue{
				{Name: "-var-file", Value: "cool.tfvars"},
			},
		},
		"both": {
			args: []string{
				"-var", "foo=bar",
				"-var-file", "cool.tfvars",
				"-var", "boop=beep",
				"replace.tfstate",
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
			got, diags := ParseStatePush(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if vars := got.Vars.All(); !cmp.Equal(vars, tc.want) {
				t.Fatalf("unexpected vars: %#v", vars)
			}
		})
	}
}

func TestParseStatePush_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *StatePush
		wantDiags tfdiags.Diagnostics
	}{
		"no arguments": {
			nil,
			&StatePush{
				Vars:      &Vars{},
				StateLock: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"Exactly one argument expected: the path to a Terraform state file.",
				),
			},
		},
		"too many arguments": {
			[]string{"foo.tfstate", "bar.tfstate"},
			&StatePush{
				Vars:      &Vars{},
				StateLock: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Required argument missing",
					"Exactly one argument expected: the path to a Terraform state file.",
				),
			},
		},
		"unknown flag": {
			[]string{"-boop"},
			&StatePush{
				Vars:      &Vars{},
				StateLock: true,
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
					"Exactly one argument expected: the path to a Terraform state file.",
				),
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseStatePush(tc.args)
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

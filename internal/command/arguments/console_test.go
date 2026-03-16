// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseConsole_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Console
	}{
		"defaults": {
			nil,
			&Console{
				Vars:         &Vars{},
				InputEnabled: true,
			},
		},
		"state flag": {
			[]string{"-state", "mystate.tfstate"},
			&Console{
				Vars:         &Vars{},
				StatePath:    "mystate.tfstate",
				InputEnabled: true,
			},
		},
		"plan flag": {
			[]string{"-plan"},
			&Console{
				Vars:         &Vars{},
				EvalFromPlan: true,
				InputEnabled: true,
			},
		},
		"input disabled": {
			[]string{"-input=false"},
			&Console{
				Vars:         &Vars{},
				InputEnabled: false,
			},
		},
		"compact warnings": {
			[]string{"-compact-warnings"},
			&Console{
				Vars:            &Vars{},
				InputEnabled:    true,
				CompactWarnings: true,
			},
		},
		"all flags": {
			[]string{"-state", "mystate.tfstate", "-plan", "-input=false", "-compact-warnings"},
			&Console{
				Vars:            &Vars{},
				StatePath:       "mystate.tfstate",
				EvalFromPlan:    true,
				InputEnabled:    false,
				CompactWarnings: true,
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseConsole(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Errorf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseConsole_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Console
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop"},
			&Console{
				Vars:         &Vars{},
				InputEnabled: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -boop",
				),
			},
		},
		"positional argument": {
			[]string{"./mydir"},
			&Console{
				Vars:         &Vars{},
				InputEnabled: true,
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Too many command line arguments",
					"The console command does not expect any positional arguments. Did you mean to use -chdir?",
				),
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseConsole(tc.args)
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Errorf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

func TestParseConsole_vars(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want []FlagNameValue
	}{
		"no var flags by default": {
			args: nil,
			want: nil,
		},
		"one var": {
			args: []string{"-var", "foo=bar"},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
			},
		},
		"one var-file": {
			args: []string{"-var-file", "cool.tfvars"},
			want: []FlagNameValue{
				{Name: "-var-file", Value: "cool.tfvars"},
			},
		},
		"ordering preserved": {
			args: []string{
				"-var", "foo=bar",
				"-var-file", "cool.tfvars",
				"-var", "boop=beep",
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
			got, diags := ParseConsole(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if vars := got.Vars.All(); !cmp.Equal(vars, tc.want) {
				t.Fatalf("unexpected result\n%s", cmp.Diff(vars, tc.want))
			}
		})
	}
}

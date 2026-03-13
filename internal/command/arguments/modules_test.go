// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseModules_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Modules
	}{
		"default": {
			nil,
			&Modules{
				ViewType: ViewHuman,
				Vars:     &Vars{},
			},
		},
		"json": {
			[]string{"-json"},
			&Modules{
				ViewType: ViewJSON,
				Vars:     &Vars{},
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseModules(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseModules_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Modules
		wantDiags tfdiags.Diagnostics
	}{
		"invalid flag": {
			[]string{"-sauron"},
			&Modules{
				ViewType: ViewHuman,
				Vars:     &Vars{},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -sauron",
				),
			},
		},
		"too many arguments": {
			[]string{"-json", "frodo"},
			&Modules{
				ViewType: ViewJSON,
				Vars:     &Vars{},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Too many command line arguments",
					"Expected no positional arguments",
				),
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseModules(tc.args)
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

func TestParseModules_vars(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want []FlagNameValue
	}{
		"var": {
			args: []string{"-var", "foo=bar"},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
			},
		},
		"var-file": {
			args: []string{"-var-file", "cool.tfvars"},
			want: []FlagNameValue{
				{Name: "-var-file", Value: "cool.tfvars"},
			},
		},
		"both": {
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
			got, diags := ParseModules(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if vars := got.Vars.All(); !cmp.Equal(vars, tc.want) {
				t.Fatalf("unexpected vars: %#v", vars)
			}
		})
	}
}

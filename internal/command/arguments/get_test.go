// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseGet_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Get
	}{
		"defaults": {
			nil,
			&Get{
				Vars:          &Vars{},
				TestDirectory: "tests",
			},
		},
		"update": {
			[]string{"-update"},
			&Get{
				Vars:          &Vars{},
				Update:        true,
				TestDirectory: "tests",
			},
		},
		"test-directory": {
			[]string{"-test-directory", "custom-tests"},
			&Get{
				Vars:          &Vars{},
				TestDirectory: "custom-tests",
			},
		},
		"all options": {
			[]string{
				"-update",
				"-test-directory", "custom-tests",
			},
			&Get{
				Vars:          &Vars{},
				Update:        true,
				TestDirectory: "custom-tests",
			},
		},
	}

	cmpOpts := cmp.Options{cmpopts.IgnoreUnexported(Vars{})}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseGet(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestParseGet_vars(t *testing.T) {
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
			got, diags := ParseGet(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if vars := got.Vars.All(); !cmp.Equal(vars, tc.want) {
				t.Fatalf("unexpected vars: %#v", vars)
			}
		})
	}
}

func TestParseGet_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Get
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-boop"},
			&Get{
				Vars:          &Vars{},
				TestDirectory: "tests",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -boop",
				),
			},
		},
		"too many arguments": {
			[]string{"foo", "bar"},
			&Get{
				Vars:          &Vars{},
				TestDirectory: "tests",
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Too many command line arguments",
					"Expected no positional arguments. Did you mean to use -chdir?",
				),
			},
		},
	}

	cmpOpts := cmp.Options{cmpopts.IgnoreUnexported(Vars{})}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseGet(tc.args)
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n got: %#v\nwant: %#v", got, tc.want)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseGraph_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Graph
	}{
		"defaults": {
			nil,
			&Graph{
				ModuleDepth: -1,
				Vars:        &Vars{},
			},
		},
		"plan type": {
			[]string{"-type=plan"},
			&Graph{
				GraphType:   "plan",
				ModuleDepth: -1,
				Vars:        &Vars{},
			},
		},
		"apply type": {
			[]string{"-type=apply"},
			&Graph{
				GraphType:   "apply",
				ModuleDepth: -1,
				Vars:        &Vars{},
			},
		},
		"draw-cycles": {
			[]string{"-draw-cycles", "-type=plan"},
			&Graph{
				DrawCycles:  true,
				GraphType:   "plan",
				ModuleDepth: -1,
				Vars:        &Vars{},
			},
		},
		"plan file": {
			[]string{"-plan=tfplan"},
			&Graph{
				Plan:        "tfplan",
				ModuleDepth: -1,
				Vars:        &Vars{},
			},
		},
		"verbose": {
			[]string{"-verbose"},
			&Graph{
				Verbose:     true,
				ModuleDepth: -1,
				Vars:        &Vars{},
			},
		},
		"module-depth": {
			[]string{"-module-depth=2"},
			&Graph{
				ModuleDepth: 2,
				Vars:        &Vars{},
			},
		},
		"all flags": {
			[]string{"-draw-cycles", "-type=plan-destroy", "-plan=tfplan", "-verbose", "-module-depth=3"},
			&Graph{
				DrawCycles:  true,
				GraphType:   "plan-destroy",
				Plan:        "tfplan",
				Verbose:     true,
				ModuleDepth: 3,
				Vars:        &Vars{},
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseGraph(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseGraph_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *Graph
		wantDiags tfdiags.Diagnostics
	}{
		"unknown flag": {
			[]string{"-wat"},
			&Graph{
				ModuleDepth: -1,
				Vars:        &Vars{},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -wat",
				),
			},
		},
		"positional argument": {
			[]string{"extra"},
			&Graph{
				ModuleDepth: -1,
				Vars:        &Vars{},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Too many command line arguments",
					"Expected no positional arguments. Did you mean to use -chdir?",
				),
			},
		},
		"too many positional arguments": {
			[]string{"bad", "bad"},
			&Graph{
				ModuleDepth: -1,
				Vars:        &Vars{},
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

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseGraph(tc.args)
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

func TestParseGraph_vars(t *testing.T) {
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
			got, diags := ParseGraph(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if vars := got.Vars.All(); !cmp.Equal(vars, tc.want) {
				t.Fatalf("unexpected vars: %#v", vars)
			}
		})
	}
}

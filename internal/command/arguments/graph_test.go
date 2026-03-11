// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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
			},
		},
		"plan type": {
			[]string{"-type=plan"},
			&Graph{
				GraphType:   "plan",
				ModuleDepth: -1,
			},
		},
		"apply type": {
			[]string{"-type=apply"},
			&Graph{
				GraphType:   "apply",
				ModuleDepth: -1,
			},
		},
		"draw-cycles": {
			[]string{"-draw-cycles", "-type=plan"},
			&Graph{
				DrawCycles:  true,
				GraphType:   "plan",
				ModuleDepth: -1,
			},
		},
		"plan file": {
			[]string{"-plan=tfplan"},
			&Graph{
				Plan:        "tfplan",
				ModuleDepth: -1,
			},
		},
		"verbose": {
			[]string{"-verbose"},
			&Graph{
				Verbose:     true,
				ModuleDepth: -1,
			},
		},
		"module-depth": {
			[]string{"-module-depth=2"},
			&Graph{
				ModuleDepth: 2,
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
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseGraph(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
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

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseGraph(tc.args)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

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
		"all options": {
			[]string{
				"-draw-cycles",
				"-type=plan",
				"-module-depth=2",
				"-verbose",
				"-plan=my.tfplan",
				"some/path",
			},
			&Graph{
				DrawCycles:  true,
				GraphType:   "plan",
				ModuleDepth: 2,
				Verbose:     true,
				PlanPath:    "my.tfplan",
				Args:        []string{"some/path"},
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
				Args:        []string{},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -wat",
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

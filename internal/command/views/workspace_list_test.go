// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestWorkspaceListJSON(t *testing.T) {
	testCases := map[string]struct {
		selected string
		list     []string
		diags    tfdiags.Diagnostics
		wantLog  map[string]interface{}
	}{
		"success": {
			"default",
			[]string{"default", "other"},
			nil,
			map[string]interface{}{
				"format_version": "1.0",
				"workspaces": []interface{}{
					map[string]interface{}{
						"name":       "default",
						"is_current": true,
					},
					map[string]interface{}{
						"name": "other",
					},
				},
				"diagnostics": []interface{}{},
			},
		},
		"success with warning": {
			"default",
			[]string{"default", "other"},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Example warning",
					"This is an example warning message.",
				),
			},
			map[string]interface{}{
				"format_version": "1.0",
				"workspaces": []interface{}{
					map[string]interface{}{
						"name":       "default",
						"is_current": true,
					},
					map[string]interface{}{
						"name": "other",
					},
				},
				"diagnostics": []interface{}{
					map[string]interface{}{
						"severity": "warning",
						"summary":  "Example warning",
						"detail":   "This is an example warning message.",
					},
				},
			},
		},
		"error": {
			"",
			[]string{},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Example error",
					"This is an example error message.",
				),
			},
			map[string]interface{}{
				"format_version": "1.0",
				"workspaces":     []interface{}{},
				"diagnostics": []interface{}{
					map[string]interface{}{
						"severity": "error",
						"summary":  "Example error",
						"detail":   "This is an example error message.",
					},
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewWorkspaceList(arguments.ViewJSON, view)

			v.List(tc.selected, tc.list, tc.diags)

			got := done(t).Stdout()

			var result map[string]interface{}
			if err := json.Unmarshal([]byte(got), &result); err != nil {
				t.Fatal("expected to be able to unmarshal JSON, got error:", err)
			}

			// Assert contents
			if diff := cmp.Diff(tc.wantLog, result); diff != "" {
				t.Fatalf("unexpected diff in JSON output:\n%s", diff)
			}
		})
	}
}

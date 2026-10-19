// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestValidateHuman(t *testing.T) {
	testCases := map[string]struct {
		diag          tfdiags.Diagnostic
		wantSubstring string
	}{
		"success": {
			nil,
			"The configuration is valid.",
		},
		"warning": {
			tfdiags.Sourceless(
				tfdiags.Warning,
				"Your shoelaces are untied",
				"Watch out, or you'll trip!",
			),
			"The configuration is valid, but there were some validation warnings",
		},
		"error": {
			tfdiags.Sourceless(
				tfdiags.Error,
				"Configuration is missing random_pet",
				"Every configuration should have a random_pet.",
			),
			"Error: Configuration is missing random_pet",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewValidate(arguments.ViewHuman, view)

			var diags tfdiags.Diagnostics

			if tc.diag != nil {
				diags = diags.Append(tc.diag)
			}

			v.Results(diags)

			got := done(t).All()

			if !strings.Contains(got, tc.wantSubstring) {
				t.Errorf("expected output to include %q, but was:\n%s", tc.wantSubstring, got)
			}
		})
	}
}

func TestValidateJSON_Results(t *testing.T) {
	testCases := map[string]struct {
		diag tfdiags.Diagnostic
	}{
		"success": {
			nil,
		},
		"warning": {
			tfdiags.Sourceless(
				tfdiags.Warning,
				"Your shoelaces are untied",
				"Watch out, or you'll trip!",
			),
		},
		"error": {
			tfdiags.Sourceless(
				tfdiags.Error,
				"Configuration is missing random_pet",
				"Every configuration should have a random_pet.",
			),
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewValidate(arguments.ViewJSON, view)

			var diags tfdiags.Diagnostics

			if tc.diag != nil {
				diags = diags.Append(tc.diag)
			}

			v.Results(diags)

			got := done(t).All()

			// Make sure the result looks like JSON; we comprehensively test
			// the structure of this output in the command package tests.
			var result map[string]interface{}

			if err := json.Unmarshal([]byte(got), &result); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestValidateJSON_Diagnostics(t *testing.T) {
	testCases := map[string]struct {
		diag       tfdiags.Diagnostics
		wantOutput string
	}{
		"error": {
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Configuration is missing random_pet",
					"Every configuration should have a random_pet.",
				),
			},
			`{
  "format_version": "1.1",
  "valid": false,
  "error_count": 0,
  "warning_count": 0,
  "diagnostics": [
    {
      "severity": "error",
      "summary": "Configuration is missing random_pet",
      "detail": "Every configuration should have a random_pet."
    }
  ]
}
`,
		},
		"nothing rendered when there are no diags": {
			tfdiags.Diagnostics{},
			"",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewValidate(arguments.ViewJSON, view)

			v.Diagnostics(tc.diag)

			output := done(t)
			gotErr := output.Stderr()
			if diff := cmp.Diff(tc.wantOutput, gotErr); diff != "" {
				t.Errorf("unexpected output diff:\n%s", diff)
			}

			got := output.Stdout()
			if got != "" {
				t.Errorf("unexpected stdout output:\n%s", got)
			}
		})
	}
}

package views

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestValidateHuman(t *testing.T) {
	testCases := map[string]struct {
		diag          tfdiags.Diagnostic
		wantSuccess   bool
		wantSubstring string
	}{
		"success": {
			nil,
			true,
			"The configuration is valid.",
		},
		"warning": {
			tfdiags.Sourceless(
				tfdiags.Warning,
				"Your shoelaces are untied",
				"Watch out, or you'll trip!",
			),
			true,
			"The configuration is valid, but there were some validation warnings",
		},
		"error": {
			tfdiags.Sourceless(
				tfdiags.Error,
				"Configuration is missing random_pet",
				"Every configuration should have a random_pet.",
			),
			false,
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

			ret := v.Results(diags)

			if tc.wantSuccess && ret != 0 {
				t.Errorf("expected 0 return code, got %d", ret)
			} else if !tc.wantSuccess && ret != 1 {
				t.Errorf("expected 1 return code, got %d", ret)
			}

			got := done(t).All()
			if strings.Contains(got, "Success!") != tc.wantSuccess {
				t.Errorf("unexpected output:\n%s", got)
			}
			if !strings.Contains(got, tc.wantSubstring) {
				t.Errorf("expected output to include %q, but was:\n%s", tc.wantSubstring, got)
			}
		})
	}
}

func TestValidateJSON(t *testing.T) {
	testCases := map[string]struct {
		diag        tfdiags.Diagnostic
		wantSuccess bool
	}{
		"success": {
			nil,
			true,
		},
		"warning": {
			tfdiags.Sourceless(
				tfdiags.Warning,
				"Your shoelaces are untied",
				"Watch out, or you'll trip!",
			),
			true,
		},
		"error": {
			tfdiags.Sourceless(
				tfdiags.Error,
				"Configuration is missing random_pet",
				"Every configuration should have a random_pet.",
			),
			false,
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

			ret := v.Results(diags)

			if tc.wantSuccess && ret != 0 {
				t.Errorf("expected 0 return code, got %d", ret)
			} else if !tc.wantSuccess && ret != 1 {
				t.Errorf("expected 1 return code, got %d", ret)
			}

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

package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestHintIfAlreadyExists(t *testing.T) {
	// 1. No error -> No hint
	diags := tfdiags.Diagnostics{}
	diags = hintIfAlreadyExists(diags)
	if len(diags) != 0 {
		t.Errorf("Expected 0 diags, got %d", len(diags))
	}

	// 2. Error matching pattern -> Hint added
	diags = tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Error,
			"Error creating resource",
			"EntityAlreadyExists: The user already exists.",
		),
	}
	diags = hintIfAlreadyExists(diags)
	if len(diags) != 2 {
		t.Errorf("Expected 2 diags, got %d", len(diags))
	} else {
		hint := diags[1]
		if hint.Severity() != tfdiags.Warning {
			t.Errorf("Expected warning severity for hint")
		}
		if hint.Description().Summary != "Hint: Resource Conflict" {
			t.Errorf("Unexpected summary: %s", hint.Description().Summary)
		}
	}

	// 3. Error NOT matching pattern -> No hint
	diags = tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Error,
			"Error creating resource",
			"Something else went wrong.",
		),
	}
	diags = hintIfAlreadyExists(diags)
	if len(diags) != 1 {
		t.Errorf("Expected 1 diag, got %d", len(diags))
	}
}

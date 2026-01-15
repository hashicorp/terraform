package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestHintIfAlreadyExists(t *testing.T) {
	tests := []struct {
		name            string
		input           tfdiags.Diagnostics
		wantCount       int
		wantWarning     bool
		wantSummary     string
		wantMessageFrag []string
	}{
		{
			name:      "no errors yields no hint",
			input:     tfdiags.Diagnostics{},
			wantCount: 0,
		},
		{
			name: "already exists error yields hint",
			input: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Error creating resource",
					"EntityAlreadyExists: The user already exists.",
				),
			},
			wantCount:   2,
			wantWarning: true,
			wantSummary: "Hint: Resource Conflict",
			wantMessageFrag: []string{
				"moved",
				"eventual consistency",
			},
		},
		{
			name: "duplicate without already exists does not yield hint",
			input: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Error creating resource",
					"DuplicateKey: Name must be unique.",
				),
			},
			wantCount: 1,
		},
		{
			name: "unrelated error does not yield hint",
			input: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Error creating resource",
					"Something else went wrong.",
				),
			},
			wantCount: 1,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			diags := hintIfAlreadyExists(tc.input)
			if len(diags) != tc.wantCount {
				t.Fatalf("Expected %d diags, got %d", tc.wantCount, len(diags))
			}
			if !tc.wantWarning {
				return
			}

			hint := diags[len(diags)-1]
			if hint.Severity() != tfdiags.Warning {
				t.Fatalf("Expected warning severity for hint")
			}
			if tc.wantSummary != "" && hint.Description().Summary != tc.wantSummary {
				t.Fatalf("Unexpected summary: %s", hint.Description().Summary)
			}
			for _, frag := range tc.wantMessageFrag {
				if !strings.Contains(hint.Description().Detail, frag) {
					t.Fatalf("Expected hint message to include %q", frag)
				}
			}
		})
	}
}

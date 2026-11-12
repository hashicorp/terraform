// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestModulesJSON_Diagnostics(t *testing.T) {
	testCases := map[string]struct {
		diags tfdiags.Diagnostics

		wantOutput string
	}{
		"errors are rendered alongside format_version and zero-values": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Error!",
					"Fix your error!",
				),
			},
			wantOutput: `{
  "format_version": "1.1",
  "modules": [],
  "diagnostics": [
    {
      "severity": "error",
      "summary": "Error!",
      "detail": "Fix your error!"
    }
  ]
}
`,
		},
		"skip if no diags": {
			diags:      tfdiags.Diagnostics{},
			wantOutput: ``,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewModulesJSON(view)

			v.Diagnostics(
				tc.diags,
			)

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

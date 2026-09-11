// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestWorkspaceShowHuman(t *testing.T) {
	testCases := map[string]struct {
		workspace string
		diags     tfdiags.Diagnostics
		wantOut   string
		wantErr   string
	}{
		"success": {
			workspace: "default",
			wantOut:   "default\n",
		},
		"success with warning": {
			workspace: "default",
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Warning, "Example warning", "This is an example warning message."),
			},
			wantOut: "Warning: Example warning\n\nThis is an example warning message.\ndefault\n",
		},
		"error": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Error, "Example error", "This is an example error message."),
			},
			wantErr: "Error: Example error\n\nThis is an example error message.\n\n",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewWorkspaceShow(arguments.ViewHuman, view)

			v.Show(tc.workspace, tc.diags)

			output := done(t)
			if diff := cmp.Diff(strings.TrimSpace(tc.wantOut), strings.TrimSpace(output.Stdout())); diff != "" {
				t.Fatalf("unexpected stdout:\n%s", diff)
			}
			if diff := cmp.Diff(strings.TrimSpace(tc.wantErr), strings.TrimSpace(output.Stderr())); diff != "" {
				t.Fatalf("unexpected stderr:\n%s", diff)
			}
		})
	}
}

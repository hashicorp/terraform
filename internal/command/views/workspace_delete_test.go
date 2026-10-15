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

func TestWorkspaceDeleteHuman_LogWorkspaceDeletionSuccess(t *testing.T) {
	testCases := map[string]struct {
		workspace  string
		diags      tfdiags.Diagnostics
		wantStdout string
		wantStderr string
	}{
		"success": {
			workspace:  "foobar",
			diags:      nil,
			wantStdout: "\nDeleted workspace \"foobar\"!",
			wantStderr: "",
		},
		"success with warning": {
			workspace: "foobar",
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Example warning",
					"This is an example warning message.",
				),
			},
			wantStdout: "Warning: Example warning\n\nThis is an example warning message.\nDeleted workspace \"foobar\"!",
			wantStderr: "",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewWorkspaceDelete(arguments.ViewHuman, view)

			v.LogWorkspaceDeletionSuccess(tc.workspace, tc.diags)

			output := done(t)

			// Assert contents
			// This must be done separately for stdout and stderr due to
			// the interleaving of output caused in tests by (TestOutput).All()
			gotStdout := strings.TrimSpace(output.Stdout())
			wantStdout := strings.TrimSpace(tc.wantStdout)
			if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
			gotStderr := strings.TrimSpace(output.Stderr())
			wantStderr := strings.TrimSpace(tc.wantStderr)
			if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
		})
	}
}

func TestWorkspaceDeleteHuman_Diagnostics(t *testing.T) {
	testCases := map[string]struct {
		diags   tfdiags.Diagnostics
		wantLog string
	}{
		"error": {
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Example error",
					"This is an example error message.",
				),
			},
			"Error: Example error\n\nThis is an example error message.\n\n",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewWorkspaceDelete(arguments.ViewHuman, view)

			v.Diagnostics(tc.diags)

			got := strings.TrimSpace(done(t).All())
			want := strings.TrimSpace(tc.wantLog)

			// Assert contents
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
		})
	}
}

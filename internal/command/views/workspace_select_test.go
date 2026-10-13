// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestWorkspaceSelectHuman_SelectedWorkspace(t *testing.T) {
	successMsg := "Switched to workspace \"default\"."

	testCases := map[string]struct {
		workspace string
		diags     tfdiags.Diagnostics
		wantOut   string
		wantErr   string
	}{
		"success": {
			workspace: "default",
			wantOut:   successMsg,
		},
		"success with warning": {
			workspace: "default",
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Warning, "Example warning", "This is an example warning message."),
			},
			wantOut: "Warning: Example warning\n\nThis is an example warning message.\n" + successMsg + "\n",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewWorkspaceSelect(arguments.ViewHuman, view)

			v.SelectedWorkspace(tc.workspace, tc.diags)

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

func TestWorkspaceSelectHuman_CreatedWorkspace(t *testing.T) {
	successMsgSnippet := fmt.Sprintf("Created and switched to workspace %q!", "default")

	testCases := map[string]struct {
		workspace string
		diags     tfdiags.Diagnostics
		wantOut   []string
		wantErr   string
	}{
		"success": {
			workspace: "default",
			wantOut:   []string{successMsgSnippet},
		},
		"success with warning": {
			workspace: "default",
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Warning, "Example warning", "This is an example warning message."),
			},
			wantOut: []string{
				"Warning: Example warning\n\nThis is an example warning message.\n",
				successMsgSnippet,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewWorkspaceSelect(arguments.ViewHuman, view)

			v.CreatedWorkspace(tc.workspace, tc.diags)

			output := done(t)
			for _, snippet := range tc.wantOut {
				if !strings.Contains(output.Stdout(), snippet) {
					t.Fatalf("expected stdout to contain snippet:\n%s", snippet)
				}
			}
			if diff := cmp.Diff(strings.TrimSpace(tc.wantErr), strings.TrimSpace(output.Stderr())); diff != "" {
				t.Fatalf("unexpected stderr:\n%s", diff)
			}
		})
	}
}

func TestWorkspaceSelectHuman_Diagnostics(t *testing.T) {
	testCases := map[string]struct {
		diags   tfdiags.Diagnostics
		wantErr string
	}{
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
			v := NewWorkspaceSelect(arguments.ViewHuman, view)

			v.Diagnostics(tc.diags)

			output := done(t)
			if output.Stdout() != "" {
				t.Fatalf("unexpected stdout:\n%s", output.Stdout())
			}
			if diff := cmp.Diff(strings.TrimSpace(tc.wantErr), strings.TrimSpace(output.Stderr())); diff != "" {
				t.Fatalf("unexpected stderr:\n%s", diff)
			}
		})
	}
}

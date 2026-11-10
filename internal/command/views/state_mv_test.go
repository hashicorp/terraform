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

func TestStateMvHuman_Move(t *testing.T) {
	testCases := map[string]struct {
		moves   []StateMvPair
		dryrun  bool
		diags   tfdiags.Diagnostics
		wantOut string
		wantErr string
	}{
		"no moves": {
			moves:   []StateMvPair{},
			wantOut: "No matching objects found.",
		},
		"no moves, dryrun": {
			moves:   []StateMvPair{},
			dryrun:  true,
			wantOut: "Would have moved nothing.",
		},
		"2 moves": {
			moves: []StateMvPair{
				{From: "foo1", To: "fez1"},
				{From: "foo2", To: "fez2"},
			},
			wantOut: "Move \"foo1\" to \"fez1\"\nMove \"foo2\" to \"fez2\"\nSuccessfully moved 2 object(s).",
		},
		"2 moves, dryrun": {
			moves: []StateMvPair{
				{From: "foo1", To: "fez1"},
				{From: "foo2", To: "fez2"},
			},
			dryrun:  true,
			wantOut: "Would move \"foo1\" to \"fez1\"\nWould move \"foo2\" to \"fez2\"",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewStateMv(arguments.ViewHuman, view)

			v.Move(tc.moves, tc.dryrun, tc.diags)

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

func TestStateMvHuman_Diagnostics(t *testing.T) {
	testCases := map[string]struct {
		workspace string
		diags     tfdiags.Diagnostics
		wantOut   string
		wantErr   string
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
			v := NewStateMv(arguments.ViewHuman, view)

			v.Diagnostics(tc.diags)

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

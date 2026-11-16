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

func TestStateRmHuman_Remove(t *testing.T) {
	testCases := map[string]struct {
		removals []string
		dryrun   bool
		diags    tfdiags.Diagnostics
		wantOut  string
		wantErr  string
	}{
		/*
		 No test case for no removals and -dryrun=false
		*/
		"no removals, dryrun": {
			removals: []string{},
			dryrun:   true,
			wantOut:  "Would have removed nothing.",
		},
		"2 removals": {
			removals: []string{"foo1", "foo2"},
			wantOut:  "Removed foo1\nRemoved foo2\nSuccessfully removed 2 resource instance(s).",
		},
		"2 removals, dryrun": {
			removals: []string{"foo1", "foo2"},
			dryrun:   true,
			wantOut:  "Would remove foo1\nWould remove foo2",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewStateRm(arguments.ViewHuman, view)

			v.Remove(tc.removals, tc.dryrun, tc.diags)

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

func TestStateRmHuman_Diagnostics(t *testing.T) {
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
			v := NewStateRm(arguments.ViewHuman, view)

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

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

func TestGraphHuman_Graph(t *testing.T) {
	graphStr := "literally anything can be passed in; command package has string builder logic"

	testCases := map[string]struct {
		diags      tfdiags.Diagnostics
		wantStdout string
	}{
		"success": {
			diags:      nil,
			wantStdout: graphStr,
		},
		"success with warning": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Example warning",
					"This is an example warning message.",
				),
			},
			wantStdout: fmt.Sprintf("Warning: Example warning\n\nThis is an example warning message.\n%s", graphStr),
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewGraph(arguments.ViewHuman, view)

			v.Graph(graphStr, tc.diags)

			output := done(t)

			// Assert contents
			gotStdout := strings.TrimSpace(output.Stdout())
			wantStdout := strings.TrimSpace(tc.wantStdout)
			if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}

			gotStderr := strings.TrimSpace(output.Stderr())
			wantStderr := ""
			if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
		})
	}
}

func TestGraphHuman_Diagnostics(t *testing.T) {
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
			"Error: Example error\n\nThis is an example error message.",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewGraph(arguments.ViewHuman, view)

			v.Diagnostics(tc.diags)

			// Assert contents
			output := done(t)
			gotStdout := strings.TrimSpace(output.Stdout())
			want := ""
			if diff := cmp.Diff(want, gotStdout); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
			gotStderr := strings.TrimSpace(output.Stderr())
			if diff := cmp.Diff(tc.wantLog, gotStderr); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
		})
	}
}

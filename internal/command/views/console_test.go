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

func TestConsoleHuman_Output(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewConsole(arguments.ViewHuman, view)

	v.Output("example output")

	output := done(t)
	expectedStdout := "example output"
	if diff := cmp.Diff(expectedStdout, strings.TrimSpace(output.Stdout())); diff != "" {
		t.Fatalf("unexpected stdout diff:\n%s", diff)
	}
	expectedStderr := ""
	if diff := cmp.Diff(expectedStderr, strings.TrimSpace(output.Stderr())); diff != "" {
		t.Fatalf("unexpected stderr diff:\n%s", diff)
	}
}

func TestConsoleHuman_Diagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewConsole(arguments.ViewHuman, view)

	diags := tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Error,
			"Example error.",
			"This is an example error detail.",
		),
	}
	v.Diagnostics(diags)

	output := done(t)
	expectedStdout := ""
	if diff := cmp.Diff(expectedStdout, strings.TrimSpace(output.Stdout())); diff != "" {
		t.Fatalf("unexpected stdout diff:\n%s", diff)
	}
	expectedStderr := "Error: Example error.\n\nThis is an example error detail."
	if diff := cmp.Diff(expectedStderr, strings.TrimSpace(output.Stderr())); diff != "" {
		t.Fatalf("unexpected stderr diff:\n%s", diff)
	}
}

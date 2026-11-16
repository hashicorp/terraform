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

func TestFmtHuman_FileFormatted(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewFmt(arguments.ViewHuman, view)

	input := "main.tf\n"
	v.FileFormatted(input)

	output := done(t)
	if diff := cmp.Diff(strings.TrimSpace(input), strings.TrimSpace(output.Stdout())); diff != "" {
		t.Fatalf("unexpected stdout:\n%s", diff)
	}
	if diff := cmp.Diff("", strings.TrimSpace(output.Stderr())); diff != "" {
		t.Fatalf("unexpected stderr:\n%s", diff)
	}
}

func TestFmtHuman_Diff(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewFmt(arguments.ViewHuman, view)

	input := []byte("i'm a diff")
	v.Diff(input)

	output := done(t)
	if diff := cmp.Diff(strings.TrimSpace(string(input)), strings.TrimSpace(output.Stdout())); diff != "" {
		t.Fatalf("unexpected stdout:\n%s", diff)
	}
	if diff := cmp.Diff("", strings.TrimSpace(output.Stderr())); diff != "" {
		t.Fatalf("unexpected stderr:\n%s", diff)
	}
}

func TestFmtHuman_Diagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewFmt(arguments.ViewHuman, view)

	v.Diagnostics(tfdiags.Diagnostics{
		tfdiags.Sourceless(tfdiags.Error, "Example error", "Error detail."),
	})

	output := done(t)
	if !strings.Contains(output.Stderr(), "Example error") {
		t.Fatalf("expected diagnostic on stderr, got %q", output.Stderr())
	}
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestStateMvHuman_diagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewStateMv(view)

	diags := tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Error,
			"Some error",
			"Some error details.",
		),
	}
	v.Diagnostics(diags)

	got := done(t).Stderr()
	if !strings.Contains(got, "Error: Some error") {
		t.Errorf("expected error output, got:\n%s", got)
	}
	if !strings.Contains(got, "Some error details.") {
		t.Errorf("expected error details, got:\n%s", got)
	}
}

func TestStateMvHuman_helpPrompt(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewStateMv(view)

	v.HelpPrompt()

	got := done(t).Stderr()
	if !strings.Contains(got, "terraform state mv -help") {
		t.Errorf("expected help prompt for 'state mv', got:\n%s", got)
	}
}

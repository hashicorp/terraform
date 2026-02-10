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

func TestImportHuman_success(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewImport(view)

	v.Success()

	got := done(t).Stdout()
	if !strings.Contains(got, "Import successful!") {
		t.Errorf("expected success message, got:\n%s", got)
	}
}

func TestImportHuman_diagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewImport(view)

	diags := tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Error,
			"Some error",
			"Some detail",
		),
	}

	v.Diagnostics(diags)

	got := done(t).Stderr()
	if !strings.Contains(got, "Some error") {
		t.Errorf("expected error in stderr, got:\n%s", got)
	}
}

func TestImportHuman_helpPrompt(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewImport(view)

	v.HelpPrompt()

	got := done(t).Stderr()
	if !strings.Contains(got, "terraform import -help") {
		t.Errorf("expected help prompt, got:\n%s", got)
	}
}

func TestImportHuman_missingResourceConfig(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewImport(view)

	v.MissingResourceConfig("test_instance.foo", "the root module", "test_instance", "foo")

	got := done(t).Stderr()
	if !strings.Contains(got, `resource address "test_instance.foo" does not exist`) {
		t.Errorf("expected missing resource error, got:\n%s", got)
	}
	if !strings.Contains(got, `resource "test_instance" "foo"`) {
		t.Errorf("expected example config, got:\n%s", got)
	}
}

func TestImportHuman_invalidAddressReference(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewImport(view)

	v.InvalidAddressReference()

	got := done(t).Stdout()
	if !strings.Contains(got, "resource-addressing") {
		t.Errorf("expected address reference, got:\n%s", got)
	}
}

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestImportHuman_MissingResourceBlockError(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewImport(arguments.ViewHuman, view)

	resource := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "aws_instance",
		Name: "example",
	}
	addr := addrs.AbsResourceInstance{
		Resource: addrs.ResourceInstance{
			Resource: resource,
			Key:      addrs.NoKey,
		},
	}
	modulePath := "module.example"

	v.MissingResourceBlockError(addr, modulePath, resource)

	output := done(t)
	expectedStdout := ""
	if diff := cmp.Diff(expectedStdout, strings.TrimSpace(output.Stdout())); diff != "" {
		t.Fatalf("unexpected stdout diff:\n%s", diff)
	}

	// We're logging to stderr for this method, see interface's code comments
	expectedStderr := `Error: resource address "aws_instance.example" does not exist in the configuration.

Before importing this resource, please create its configuration in module.example. For example:

resource "aws_instance" "example" {
  # (resource arguments)
}`
	if diff := cmp.Diff(expectedStderr, strings.TrimSpace(output.Stderr())); diff != "" {
		t.Fatalf("unexpected stderr diff:\n%s", diff)
	}
}

func TestImportHuman_Success(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewImport(arguments.ViewHuman, view)

	v.Success()

	output := done(t)
	expectedStdoutSnippet := "Import successful!"
	if !strings.Contains(strings.TrimSpace(output.Stdout()), expectedStdoutSnippet) {
		t.Fatalf("expected stdout to contain %q, got %q", expectedStdoutSnippet, strings.TrimSpace(output.Stdout()))
	}
	expectedStderr := ""
	if diff := cmp.Diff(expectedStderr, strings.TrimSpace(output.Stderr())); diff != "" {
		t.Fatalf("unexpected stderr diff:\n%s", diff)
	}
}

func TestImportHuman_InvalidAddressHelp(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewImport(arguments.ViewHuman, view)

	v.InvalidAddressHelp()

	output := done(t)
	expectedStdoutSnippet := "For information on valid syntax, see:"
	if !strings.Contains(strings.TrimSpace(output.Stdout()), expectedStdoutSnippet) {
		t.Fatalf("expected stdout to contain %q, got %q", expectedStdoutSnippet, strings.TrimSpace(output.Stdout()))
	}
	expectedStderr := ""
	if diff := cmp.Diff(expectedStderr, strings.TrimSpace(output.Stderr())); diff != "" {
		t.Fatalf("unexpected stderr diff:\n%s", diff)
	}
}

func TestImportHuman_Diagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewImport(arguments.ViewHuman, view)

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

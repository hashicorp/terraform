// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestGetHuman_LogModuleDownload(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewGet(arguments.ViewHuman, view)

	message := "foobar" // currently this method just echoes the message it receives
	v.LogModuleDownload(message)

	output := done(t)
	if diff := cmp.Diff(strings.TrimSpace(message), strings.TrimSpace(output.Stdout())); diff != "" {
		t.Fatalf("unexpected stdout diff:\n%s", diff)
	}
	if diff := cmp.Diff(strings.TrimSpace(""), strings.TrimSpace(output.Stderr())); diff != "" {
		t.Fatalf("unexpected stderr diff:\n%s", diff)
	}
}

func TestGetHuman_LogModuleInstallation(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewGet(arguments.ViewHuman, view)

	message := "foobar" // currently this method just echoes the message it receives
	v.LogModuleInstallation(message)

	output := done(t)
	if diff := cmp.Diff(strings.TrimSpace(message), strings.TrimSpace(output.Stdout())); diff != "" {
		t.Fatalf("unexpected stdout diff:\n%s", diff)
	}
	if diff := cmp.Diff(strings.TrimSpace(""), strings.TrimSpace(output.Stderr())); diff != "" {
		t.Fatalf("unexpected stderr diff:\n%s", diff)
	}
}

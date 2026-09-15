// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestGetHuman_LogModuleDownload(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewGet(arguments.ViewHuman, view)

	packageAddr := "registry.terraform.io/hashicorp/consul"
	version := (*version.Version)(nil)
	modulePath := "module.network"
	v.LogModuleDownload(packageAddr, version, modulePath)

	output := done(t)
	expected := "Downloading registry.terraform.io/hashicorp/consul for module.network..."
	if diff := cmp.Diff(expected, strings.TrimSpace(output.Stdout())); diff != "" {
		t.Fatalf("unexpected stdout diff:\n%s", diff)
	}
	if diff := cmp.Diff(strings.TrimSpace(""), strings.TrimSpace(output.Stderr())); diff != "" {
		t.Fatalf("unexpected stderr diff:\n%s", diff)
	}
}

func TestGetHuman_LogModuleInstallation(t *testing.T) {
	t.Run("show local paths", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		view.Configure(&arguments.View{NoColor: true})
		v := NewGet(arguments.ViewHuman, view)

		modulePath := "module.network"
		localDir := "/local/dir"
		v.LogModuleInstallation(modulePath, localDir)

		output := done(t)
		expected := "- module.network in /local/dir"
		if diff := cmp.Diff(expected, strings.TrimSpace(output.Stdout())); diff != "" {
			t.Fatalf("unexpected stdout diff:\n%s", diff)
		}
		if diff := cmp.Diff(strings.TrimSpace(""), strings.TrimSpace(output.Stderr())); diff != "" {
			t.Fatalf("unexpected stderr diff:\n%s", diff)
		}
	})

	t.Run("don't show local paths", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		view.Configure(&arguments.View{NoColor: true})
		v := NewGet(arguments.ViewHuman, view)

		modulePath := "module.network"
		localDir := ""
		v.LogModuleInstallation(modulePath, localDir)

		output := done(t)
		expected := "- module.network"
		if diff := cmp.Diff(expected, strings.TrimSpace(output.Stdout())); diff != "" {
			t.Fatalf("unexpected stdout diff:\n%s", diff)
		}
		if diff := cmp.Diff(strings.TrimSpace(""), strings.TrimSpace(output.Stderr())); diff != "" {
			t.Fatalf("unexpected stderr diff:\n%s", diff)
		}
	})
}

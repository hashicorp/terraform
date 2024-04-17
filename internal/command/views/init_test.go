// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
)

func TestNewInit_jsonViewDiagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)

	newInit := NewInit(arguments.ViewJSON, NewView(streams).SetRunningInAutomation(true))
	if _, ok := newInit.(*InitJSON); !ok {
		t.Fatalf("unexpected return type %t", newInit)
	}

	diags := getTestDiags(t)
	newInit.Diagnostics(diags)

	version := tfversion.String()
	want := []map[string]interface{}{
		{
			"@level":    "info",
			"@message":  fmt.Sprintf("Terraform %s", version),
			"@module":   "terraform.ui",
			"terraform": version,
			"type":      "version",
			"ui":        JSON_UI_VERSION,
		},
		{
			"@level":   "error",
			"@message": "Error: Error selecting workspace",
			"@module":  "terraform.ui",
			"diagnostic": map[string]interface{}{
				"severity": "error",
				"summary":  "Error selecting workspace",
				"detail":   "Workspace random_pet does not exist",
			},
			"type": "diagnostic",
		},
		{
			"@level":   "error",
			"@message": "Error: Unsupported backend type",
			"@module":  "terraform.ui",
			"diagnostic": map[string]interface{}{
				"severity": "error",
				"summary":  "Unsupported backend type",
				"detail":   "There is no explicit backend type named fake backend.",
			},
			"type": "diagnostic",
		},
	}

	actual := done(t).Stdout()
	testJSONViewOutputEqualsFull(t, actual, want)
}

func TestNewInit_humanViewDiagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)

	newInit := NewInit(arguments.ViewHuman, NewView(streams).SetRunningInAutomation(true))
	if _, ok := newInit.(*InitHuman); !ok {
		t.Fatalf("unexpected return type %t", newInit)
	}

	diags := getTestDiags(t)
	newInit.Diagnostics(diags)

	actual := done(t).All()
	expected := "\nError: Error selecting workspace\n\nWorkspace random_pet does not exist\n\nError: Unsupported backend type\n\nThere is no explicit backend type named fake backend.\n"
	if !strings.Contains(actual, expected) {
		t.Fatalf("expected output to contain: %s, but got %s", expected, actual)
	}
}

func TestNewInit_unsupportedViewDiagnostics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("should panic with unsupported view type raw")
		} else if r != "unknown view type raw" {
			t.Fatalf("unexpected panic message: %v", r)
		}
	}()

	streams, done := terminal.StreamsForTesting(t)
	defer done(t)

	NewInit(arguments.ViewRaw, NewView(streams).SetRunningInAutomation(true))
}

func getTestDiags(t *testing.T) tfdiags.Diagnostics {
	t.Helper()

	var diags tfdiags.Diagnostics
	diags = diags.Append(
		tfdiags.Sourceless(
			tfdiags.Error,
			"Error selecting workspace",
			"Workspace random_pet does not exist",
		),
		&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported backend type",
			Detail:   "There is no explicit backend type named fake backend.",
			Subject:  nil,
		},
	)

	return diags
}

func TestNewInit_jsonViewOutput(t *testing.T) {
	t.Run("no param", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)

		newInit := NewInit(arguments.ViewJSON, NewView(streams).SetRunningInAutomation(true))
		if _, ok := newInit.(*InitJSON); !ok {
			t.Fatalf("unexpected return type %t", newInit)
		}

		newInit.Output(InitializingProviderPluginMessage)

		version := tfversion.String()
		want := []map[string]interface{}{
			{
				"@level":    "info",
				"@message":  fmt.Sprintf("Terraform %s", version),
				"@module":   "terraform.ui",
				"terraform": version,
				"type":      "version",
				"ui":        JSON_UI_VERSION,
			},
			{
				"@level":       "info",
				"@message":     "Initializing provider plugins...",
				"message_code": "initializing_provider_plugin_message",
				"@module":      "terraform.ui",
				"type":         "init_output",
			},
		}

		actual := done(t).Stdout()
		testJSONViewOutputEqualsFull(t, actual, want)
	})

	t.Run("single param", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)

		newInit := NewInit(arguments.ViewJSON, NewView(streams).SetRunningInAutomation(true))
		if _, ok := newInit.(*InitJSON); !ok {
			t.Fatalf("unexpected return type %t", newInit)
		}

		packageName := "hashicorp/aws"
		newInit.Output(FindingLatestVersionMessage, packageName)

		version := tfversion.String()
		want := []map[string]interface{}{
			{
				"@level":    "info",
				"@message":  fmt.Sprintf("Terraform %s", version),
				"@module":   "terraform.ui",
				"terraform": version,
				"type":      "version",
				"ui":        JSON_UI_VERSION,
			},
			{
				"@level":       "info",
				"@message":     fmt.Sprintf("%s: Finding latest version...", packageName),
				"@module":      "terraform.ui",
				"message_code": "finding_latest_version_message",
				"type":         "init_output",
			},
		}

		actual := done(t).Stdout()
		testJSONViewOutputEqualsFull(t, actual, want)
	})

	t.Run("variable length params", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)

		newInit := NewInit(arguments.ViewJSON, NewView(streams).SetRunningInAutomation(true))
		if _, ok := newInit.(*InitJSON); !ok {
			t.Fatalf("unexpected return type %t", newInit)
		}

		var packageName, packageVersion = "hashicorp/aws", "3.0.0"
		newInit.Output(ProviderAlreadyInstalledMessage, packageName, packageVersion)

		version := tfversion.String()
		want := []map[string]interface{}{
			{
				"@level":    "info",
				"@message":  fmt.Sprintf("Terraform %s", version),
				"@module":   "terraform.ui",
				"terraform": version,
				"type":      "version",
				"ui":        JSON_UI_VERSION,
			},
			{
				"@level":       "info",
				"@message":     fmt.Sprintf("%s v%s: Using previously-installed provider version", packageName, packageVersion),
				"@module":      "terraform.ui",
				"message_code": "provider_already_installed_message",
				"type":         "init_output",
			},
		}

		actual := done(t).Stdout()
		testJSONViewOutputEqualsFull(t, actual, want)
	})
}

func TestNewInit_jsonViewLog(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)

	newInit := NewInit(arguments.ViewJSON, NewView(streams).SetRunningInAutomation(true))
	if _, ok := newInit.(*InitJSON); !ok {
		t.Fatalf("unexpected return type %t", newInit)
	}

	newInit.LogInitMessage(InitializingProviderPluginMessage)

	version := tfversion.String()
	want := []map[string]interface{}{
		{
			"@level":    "info",
			"@message":  fmt.Sprintf("Terraform %s", version),
			"@module":   "terraform.ui",
			"terraform": version,
			"type":      "version",
			"ui":        JSON_UI_VERSION,
		},
		{
			"@level":   "info",
			"@message": "Initializing provider plugins...",
			"@module":  "terraform.ui",
			"type":     "log",
		},
	}

	actual := done(t).Stdout()
	testJSONViewOutputEqualsFull(t, actual, want)
}

func TestNewInit_jsonViewPrepareMessage(t *testing.T) {
	t.Run("existing message code", func(t *testing.T) {
		streams, _ := terminal.StreamsForTesting(t)

		newInit := NewInit(arguments.ViewJSON, NewView(streams).SetRunningInAutomation(true))
		if _, ok := newInit.(*InitJSON); !ok {
			t.Fatalf("unexpected return type %t", newInit)
		}

		want := "Initializing modules..."

		actual := newInit.PrepareMessage(InitializingModulesMessage)
		if !cmp.Equal(want, actual) {
			t.Errorf("unexpected output: %s", cmp.Diff(want, actual))
		}
	})
}

func TestNewInit_humanViewOutput(t *testing.T) {
	t.Run("no param", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)

		newInit := NewInit(arguments.ViewHuman, NewView(streams).SetRunningInAutomation(true))
		if _, ok := newInit.(*InitHuman); !ok {
			t.Fatalf("unexpected return type %t", newInit)
		}

		newInit.Output(InitializingProviderPluginMessage)

		actual := done(t).All()
		expected := "Initializing provider plugins..."
		if !strings.Contains(actual, expected) {
			t.Fatalf("expected output to contain: %s, but got %s", expected, actual)
		}
	})

	t.Run("single param", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)

		newInit := NewInit(arguments.ViewHuman, NewView(streams).SetRunningInAutomation(true))
		if _, ok := newInit.(*InitHuman); !ok {
			t.Fatalf("unexpected return type %t", newInit)
		}

		packageName := "hashicorp/aws"
		newInit.Output(FindingLatestVersionMessage, packageName)

		actual := done(t).All()
		expected := "Finding latest version of hashicorp/aws"
		if !strings.Contains(actual, expected) {
			t.Fatalf("expected output to contain: %s, but got %s", expected, actual)
		}
	})

	t.Run("variable length params", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)

		newInit := NewInit(arguments.ViewHuman, NewView(streams).SetRunningInAutomation(true))
		if _, ok := newInit.(*InitHuman); !ok {
			t.Fatalf("unexpected return type %t", newInit)
		}

		var packageName, packageVersion = "hashicorp/aws", "3.0.0"
		newInit.Output(ProviderAlreadyInstalledMessage, packageName, packageVersion)

		actual := done(t).All()
		expected := "- Using previously-installed hashicorp/aws v3.0.0"
		if !strings.Contains(actual, expected) {
			t.Fatalf("expected output to contain: %s, but got %s", expected, actual)
		}
	})
}

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	viewjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
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

func TestNewInit_jsonViewPolicyResults(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)

	newInit := NewInit(arguments.ViewJSON, NewView(streams).SetRunningInAutomation(true))
	if _, ok := newInit.(*InitJSON); !ok {
		t.Fatalf("unexpected return type %t", newInit)
	}

	results := plans.NewPolicyResults()
	results.AddModule(
		addrs.RootModule.Child("example"),
		policy.EvaluationResponse{
			Overall: policy.DenyResult,
			Diagnostics: policy.Diagnostics{
				policy.NewErrorDiagnostic(
					"module policy denied",
					"module policy blocked installation",
					policy.DenyResult,
				),
			},
			Policies: []*policy.Policy{
				{
					Address:          "module_policy.example",
					Filename:         "policy_file.tfpolicy.hcl",
					EnforcementLevel: "mandatory",
					Result:           policy.DenyResult,
				},
			},
		},
		nil,
	)

	newInit.PolicyResults(results)

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
			"@message": "Error: module policy denied",
			"@module":  "terraform.ui",
			"@policy":  "true",
			"policy_diagnostic": map[string]interface{}{
				"severity": "error",
				"summary":  "module policy denied",
				"detail":   "module policy blocked installation",
			},
			"policy_metadata": map[string]interface{}{},
			"result":          policy.DenyResult.String(),
			"target_address":  "module.example",
			"type":            string(viewjson.MessagePolicyDiagnostic),
		},
		{
			"@level":         "info",
			"@message":       "Policy Result",
			"@module":        "terraform.ui",
			"@policy":        "true",
			"target_address": "module.example",
			"policy_address": "module_policy.example",
			"policy_metadata": map[string]interface{}{
				"policy_name":       "module_policy.example",
				"file_name":         "policy_file.tfpolicy.hcl",
				"enforcement_level": "mandatory",
			},
			"result": policy.DenyResult.String(),
			"type":   string(viewjson.MessagePolicyEvaluationResult),
		},
	}

	actual := done(t).Stdout()
	testJSONViewOutputEqualsFull(t, actual, want)
}

func TestNewInit_humanViewPolicyResults(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)

	newInit := NewInit(arguments.ViewHuman, NewView(streams).SetRunningInAutomation(true))
	if _, ok := newInit.(*InitHuman); !ok {
		t.Fatalf("unexpected return type %t", newInit)
	}

	results := plans.NewPolicyResults()
	results.AddModule(
		addrs.RootModule.Child("example"),
		policy.EvaluationResponse{
			Overall: policy.DenyResult,
			Diagnostics: policy.Diagnostics{
				policy.NewErrorDiagnostic(
					"module policy denied",
					"module policy blocked installation",
					policy.DenyResult,
				),
			},
		},
		nil,
	)

	newInit.PolicyResults(results)

	actual := done(t).All()
	expected := "\nError: module policy denied\n\nmodule policy blocked installation\n"
	if !strings.Contains(actual, expected) {
		t.Fatalf("expected output to contain: %s, but got %s", expected, actual)
	}
}

func TestNewInit_humanViewPolicyResults_infoWithoutSnippet(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)

	newInit := NewInit(arguments.ViewHuman, NewView(streams).SetRunningInAutomation(true))
	if _, ok := newInit.(*InitHuman); !ok {
		t.Fatalf("unexpected return type %t", newInit)
	}

	results := plans.NewPolicyResults()
	results.AddModule(
		addrs.RootModule.Child("example"),
		policy.EvaluationResponse{
			Overall: policy.AllowResult,
			Enforcements: []policy.EnforcementResult{{
				Result:  policy.AllowResult,
				Message: "module policy allowed installation",
				Policy: &policy.Policy{
					Address: "module_policy.example",
				},
			}},
		},
		nil,
	)

	newInit.PolicyResults(results)

	actual := done(t).Stdout()
	if !strings.Contains(actual, "Policy Info:") {
		t.Fatalf("expected output to contain policy info header, but got %s", actual)
	}
	if !strings.Contains(actual, "module policy allowed installation") {
		t.Fatalf("expected output to contain policy message, but got %s", actual)
	}
	if !strings.Contains(actual, "module_policy.example") {
		t.Fatalf("expected output to contain policy address fallback, but got %s", actual)
	}
}

func TestNewInit_jsonViewOutput(t *testing.T) {
	t.Run("no param", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)

		newInit := NewInit(arguments.ViewJSON, NewView(streams).SetRunningInAutomation(true))
		if _, ok := newInit.(*InitJSON); !ok {
			t.Fatalf("unexpected return type %t", newInit)
		}

		newInit.Output(InitializingProviderPluginFromConfigMessage)

		version := tfversion.String()
		want := []map[string]any{
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
				"@message":     "Initializing provider plugins found in the configuration...",
				"message_code": "initializing_provider_plugin_from_config_message",
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

	newInit.LogInitMessage(InitializingProviderPluginFromConfigMessage)

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
			"@message": "Initializing provider plugins found in the configuration...",
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

		newInit.Output(InitializingProviderPluginFromConfigMessage)

		actual := done(t).All()
		expected := "Initializing provider plugins found in the configuration..."
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

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
)

func TestNewInit_jsonView(t *testing.T) {
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

func TestNewInit_humanView(t *testing.T) {
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

func TestNewInit_unsupportedView(t *testing.T) {
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

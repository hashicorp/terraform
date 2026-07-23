// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"
	"testing"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	viewjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/getproviders"
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

	newInit.PolicyResult(
		addrs.RootModule.Child("example").String(),
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
	)

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

	newInit.PolicyResult(
		addrs.RootModule.Child("example").String(),
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
	)

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

	newInit.PolicyResult(
		addrs.RootModule.Child("example").String(),
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
	)

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

		newInit.Output(InitializingProviderPluginMessage)

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

		packageName, packageVersion := "hashicorp/aws", "3.0.0"
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
			"@module":      "terraform.ui",
			"message_code": "initializing_provider_plugin_message",
			"type":         "init_output",
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

		actual := newInit.prepareMessage(InitializingModulesMessage)
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

		packageName, packageVersion := "hashicorp/aws", "3.0.0"
		newInit.Output(ProviderAlreadyInstalledMessage, packageName, packageVersion)

		actual := done(t).All()
		expected := "- Using previously-installed hashicorp/aws v3.0.0"
		if !strings.Contains(actual, expected) {
			t.Fatalf("expected output to contain: %s, but got %s", expected, actual)
		}
	})
}

// Assert message content
func TestNewInit_LogProviderVersionSuccess(t *testing.T) {
	const verifiedChecksum = 0
	const officialProvider = 1
	const noKey = ""

	t.Run("no auth result - human view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		initView := NewInit(arguments.ViewHuman, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		var authResult *getproviders.PackageAuthenticationResult = nil

		initView.LogProviderVersionSuccess(p, ver, authResult)

		// Assert output
		output := done(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (unauthenticated)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
	})
	t.Run("no auth result - json view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		initView := NewInit(arguments.ViewJSON, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		var authResult *getproviders.PackageAuthenticationResult = nil

		initView.LogProviderVersionSuccess(p, ver, authResult)

		// Assert output - human
		output := done(t)
		expectedOutput := `"@message":"Installed provider version: hashicorp/test v1.2.3 (unauthenticated)"`
		if !strings.Contains(output.Stdout(), expectedOutput) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", expectedOutput, output.Stdout())
		}
	})
	t.Run("verified checksum auth result - human view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		initView := NewInit(arguments.ViewHuman, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		authResult := getproviders.NewPackageAuthenticationResult(verifiedChecksum, noKey)

		initView.LogProviderVersionSuccess(p, ver, authResult)

		// Assert output
		output := done(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (verified checksum)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
	})
	t.Run("verified checksum auth result - json view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		initView := NewInit(arguments.ViewJSON, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		authResult := getproviders.NewPackageAuthenticationResult(verifiedChecksum, noKey)

		initView.LogProviderVersionSuccess(p, ver, authResult)

		// Assert output - human
		output := done(t)
		expectedOutput := `"@message":"Installed provider version: hashicorp/test v1.2.3 (verified checksum)"`
		if !strings.Contains(output.Stdout(), expectedOutput) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", expectedOutput, output.Stdout())
		}
	})
	t.Run("official provider auth result - human view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		initView := NewInit(arguments.ViewHuman, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		key := "key-id-123"
		authResult := getproviders.NewPackageAuthenticationResult(officialProvider, key)

		initView.LogProviderVersionSuccess(p, ver, authResult)

		// Assert output
		output := done(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (signed by HashiCorp)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
	})
	t.Run("official provider auth result - json view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		initView := NewInit(arguments.ViewJSON, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		key := "key-id-123"
		authResult := getproviders.NewPackageAuthenticationResult(officialProvider, key)

		initView.LogProviderVersionSuccess(p, ver, authResult)

		// Assert output - human
		output := done(t)
		expectedOutput := `"@message":"Installed provider version: hashicorp/test v1.2.3 (signed by HashiCorp)"`
		if !strings.Contains(output.Stdout(), expectedOutput) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", expectedOutput, output.Stdout())
		}
	})
}

// Assert message content
func TestNewInit_LogProviderVersionSuccessWithKeyID(t *testing.T) {
	const partnerProvider = 2

	t.Run("partner provider auth result - human view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		initView := NewInit(arguments.ViewHuman, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		key := "key-id-123"
		authResult := getproviders.NewPackageAuthenticationResult(partnerProvider, key)

		initView.LogProviderVersionSuccessWithKeyID(p, ver, authResult, key)

		// Assert output - human
		output := done(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (signed by a HashiCorp partner, key ID key-id-123)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
	})
	t.Run("partner provider auth result -json view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		initView := NewInit(arguments.ViewJSON, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		key := "key-id-123"
		authResult := getproviders.NewPackageAuthenticationResult(partnerProvider, key)

		initView.LogProviderVersionSuccessWithKeyID(p, ver, authResult, key)

		// Assert output - human
		output := done(t)
		expectedOutput := `{"@level":"info","@message":"Installed provider version: hashicorp/test v1.2.3 (signed by a HashiCorp partnerkey_id: key-id-123)","@module":"terraform.ui","@timestamp":` // Stop comparison before timestamp
		if !strings.Contains(output.Stdout(), expectedOutput) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", expectedOutput, output.Stdout())
		}
	})
}

// Assert JSON log content, including log type and additional fields
func TestNewInit_LogProviderVersionSuccess_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	initView := NewInit(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	v := versions.MustParseVersion("1.0.0")
	officialProvider := 1
	authResult := getproviders.NewPackageAuthenticationResult(officialProvider, "key-id-123")
	initView.LogProviderVersionSuccess(p, v, authResult)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Installed provider version: hashicorp/test v1.0.0 (signed by HashiCorp)"`,
		`"@module":"terraform.ui"`,
		`"type":"log"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewInit_LogProviderVersionAlreadyInstalled_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	initView := NewInit(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	v := versions.MustParseVersion("1.0.0")
	initView.LogProviderVersionAlreadyInstalled(p, v)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"hashicorp/test v1.0.0: Using previously-installed provider version"`,
		`"@module":"terraform.ui"`,
		`"type":"log"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

// Assert JSON log content, including log type and additional fields
//
// Note - in calling code this is only ever used for partner providers
func TestNewInit_LogProviderVersionSuccessWithKeyID_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	initView := NewInit(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	v := versions.MustParseVersion("1.0.0")
	partnerProvider := 2
	keyID := "key-id-123"
	authResult := getproviders.NewPackageAuthenticationResult(partnerProvider, keyID)
	initView.LogProviderVersionSuccessWithKeyID(p, v, authResult, keyID)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Installed provider version: hashicorp/test v1.0.0 (signed by a HashiCorp partnerkey_id: key-id-123)"`,
		`"@module":"terraform.ui"`,
		`"type":"log"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewInit_LogReusingPreviousProviderVersion_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	initView := NewInit(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	version := getproviders.MustParseVersion("1.0.0")
	initView.LogReusingPreviousProviderVersion(p, version)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Reusing version 1.0.0 of hashicorp/test from the dependency lock file"`,
		`"@module":"terraform.ui"`,
		`"type":"log"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewInit_LogFindingMatchingVersion_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	initView := NewInit(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	constraint, _ := getproviders.ParseVersionConstraints("1.0.0")
	initView.LogFindingMatchingVersion(p, constraint)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Finding matching versions for provider: hashicorp/test, version_constraint: \"1.0.0\""`,
		`"@module":"terraform.ui"`,
		`"type":"log"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewInit_LogFindingLatestVersion_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	initView := NewInit(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	initView.LogFindingLatestVersion(p)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"hashicorp/test: Finding latest version..."`,
		`"@module":"terraform.ui"`,
		`"type":"log"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewInit_LogInstallingProviderVersion_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	initView := NewInit(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	v := versions.MustParseVersion("1.0.0")
	initView.LogInstallingProviderVersion(p, v)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Installing provider version: hashicorp/test v1.0.0..."`,
		`"@module":"terraform.ui"`,
		`"type":"log"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewInit_LogBuiltInProviderAvailable_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	initView := NewInit(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	initView.LogBuiltInProviderAvailable(p)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"hashicorp/test is built in to Terraform"`,
		`"@module":"terraform.ui"`,
		`"type":"log"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewInit_LogUsingProviderVersionFromCacheDir_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	initView := NewInit(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	v := versions.MustParseVersion("1.0.0")
	initView.LogUsingProviderVersionFromCacheDir(p, v)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"hashicorp/test v1.0.0: Using from the shared cache directory"`,
		`"@module":"terraform.ui"`,
		`"type":"log"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewInit_LogPartnerAndCommunityProviders_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	initView := NewInit(arguments.ViewJSON, view)

	initView.LogPartnerAndCommunityProviders()

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Partner and community providers are signed by their developers.\nIf you'd like to know more about provider signing, you can read about it here:\nhttps://developer.hashicorp.com/terraform/cli/plugins/signing"`,
		`"@module":"terraform.ui"`,
		`"type":"log"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewInit_LogInitializingStateStoreProviderPlugin_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	initView := NewInit(arguments.ViewJSON, view)

	pAddr := addrs.NewDefaultProvider("test")
	cons := getproviders.MustParseVersionConstraints("~> 1.0")
	storeType := "test_store"
	initView.LogInitializingStateStoreProviderPlugin(pAddr, cons, storeType)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Initializing provider hashicorp/test (~\u003e 1.0) for state store \"test_store\"..."`,
		`"@module":"terraform.ui"`,
		//@timestamp is dynamic
		`"message_code":"initializing_state_store_provider_plugin_message"`,
		`"type":"init_output"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewInit_Spacer_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	initView := NewInit(arguments.ViewJSON, view)

	initView.Spacer()

	// Assert output
	output := done(t)

	// We cannot simply assert no output as the JSON view logs the version message on initialization
	// Splitting on \n when there's only the version log will get an array of the log and an empty string.
	// If there are more logs there'll be >2 elements.
	if x := strings.Split(output.Stdout(), "\n"); len(x) != 2 {
		t.Fatalf("expected no additional output after version message, got: %s", output.Stdout())
	}
}

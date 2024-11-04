package views

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
)

func TestNewCloud_unsupportedViewDiagnostics(t *testing.T) {
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

	NewCloud(arguments.ViewRaw, NewView(streams).SetRunningInAutomation(true))
}

func TestNewCloud_humanViewOutput(t *testing.T) {
	t.Run("no param", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)

		newCloud := NewCloud(arguments.ViewHuman, NewView(streams).SetRunningInAutomation(true))
		if _, ok := newCloud.(*CloudHuman); !ok {
			t.Fatalf("unexpected return type %t", newCloud)
		}

		newCloud.Output(InitialRetryErrorMessage)

		actual := done(t).All()
		expected := "There was an error connecting to HCP Terraform. Please do not exit\nTerraform to prevent data loss! Trying to restore the connection..."
		if !strings.Contains(actual, expected) {
			t.Fatalf("expected output to contain: %s, but got %s", expected, actual)
		}
	})

	t.Run("single param", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)

		newCloud := NewCloud(arguments.ViewHuman, NewView(streams).SetRunningInAutomation(true))
		if _, ok := newCloud.(*CloudHuman); !ok {
			t.Fatalf("unexpected return type %t", newCloud)
		}

		duration := 5 * time.Second
		newCloud.Output(RepeatedRetryErrorMessage, duration)

		actual := done(t).All()
		expected := fmt.Sprintf("Still trying to restore the connection... (%s elapsed)", duration)
		if !strings.Contains(actual, expected) {
			t.Fatalf("expected output to contain: %s, but got %s", expected, actual)
		}
	})
}

func TestNewCloud_humanViewPrepareMessage(t *testing.T) {
	t.Run("existing message code", func(t *testing.T) {
		streams, _ := terminal.StreamsForTesting(t)

		newCloud := NewCloud(arguments.ViewHuman, NewView(streams).SetRunningInAutomation(true))
		if _, ok := newCloud.(*CloudHuman); !ok {
			t.Fatalf("unexpected return type %t", newCloud)
		}

		want := "\nThere was an error connecting to HCP Terraform. Please do not exit\nTerraform to prevent data loss! Trying to restore the connection..."

		actual := newCloud.PrepareMessage(InitialRetryErrorMessage)
		if !cmp.Equal(want, actual) {
			t.Errorf("unexpected output: %s", cmp.Diff(want, actual))
		}
	})
}

func TestNewCloud_humanViewDiagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)

	newCloud := NewCloud(arguments.ViewHuman, NewView(streams).SetRunningInAutomation(true))
	if _, ok := newCloud.(*CloudHuman); !ok {
		t.Fatalf("unexpected return type %t", newCloud)
	}

	diags := getHCPDiags(t)
	newCloud.Diagnostics(diags)

	actual := done(t).All()
	expected := "\nError: Error connecting to HCP\n\nCould not connect to HCP Terraform. Check your network.\n\nError: Network Timeout\n\nConnection to HCP timed out. Check your network.\n"
	if !strings.Contains(actual, expected) {
		t.Fatalf("expected output to contain: %s, but got %s", expected, actual)
	}
}

func TestNewCloud_jsonViewOutput(t *testing.T) {
	t.Run("no param", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)

		newCloud := NewCloud(arguments.ViewJSON, NewView(streams).SetRunningInAutomation(true))
		if _, ok := newCloud.(*CloudJSON); !ok {
			t.Fatalf("unexpected return type %t", newCloud)
		}

		newCloud.Output(InitialRetryErrorMessage)

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
				"@message":     "There was an error connecting to HCP Terraform. Please do not exit\nTerraform to prevent data loss! Trying to restore the connection...",
				"message_code": "initial_retry_error_message",
				"@module":      "terraform.ui",
				"type":         "cloud_output",
			},
		}

		actual := done(t).Stdout()
		testJSONViewOutputEqualsFull(t, actual, want)
	})

	t.Run("single param", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)

		newCloud := NewCloud(arguments.ViewJSON, NewView(streams).SetRunningInAutomation(true))
		if _, ok := newCloud.(*CloudJSON); !ok {
			t.Fatalf("unexpected return type %t", newCloud)
		}

		duration := 5 * time.Second
		newCloud.Output(RepeatedRetryErrorMessage, duration)

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
				"@message":     fmt.Sprintf("Still trying to restore the connection... (%s elapsed)", duration),
				"@module":      "terraform.ui",
				"message_code": "repeated_retry_error_message",
				"type":         "cloud_output",
			},
		}

		actual := done(t).Stdout()
		testJSONViewOutputEqualsFull(t, actual, want)
	})
}

func TestNewCloud_jsonViewPrepareMessage(t *testing.T) {
	t.Run("existing message code", func(t *testing.T) {
		streams, _ := terminal.StreamsForTesting(t)

		newCloud := NewCloud(arguments.ViewJSON, NewView(streams).SetRunningInAutomation(true))
		if _, ok := newCloud.(*CloudJSON); !ok {
			t.Fatalf("unexpected return type %t", newCloud)
		}

		want := "There was an error connecting to HCP Terraform. Please do not exit\nTerraform to prevent data loss! Trying to restore the connection..."

		actual := newCloud.PrepareMessage(InitialRetryErrorMessage)
		if !cmp.Equal(want, actual) {
			t.Errorf("unexpected output: %s", cmp.Diff(want, actual))
		}
	})
}

func TestNewCloud_jsonViewDiagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)

	newCloud := NewCloud(arguments.ViewJSON, NewView(streams).SetRunningInAutomation(true))
	if _, ok := newCloud.(*CloudJSON); !ok {
		t.Fatalf("unexpected return type %t", newCloud)
	}

	diags := getHCPDiags(t) // Assuming you want to use the HCP diagnostics here
	newCloud.Diagnostics(diags)

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
			"@message": "Error: Error connecting to HCP",
			"@module":  "terraform.ui",
			"diagnostic": map[string]interface{}{
				"severity": "error",
				"summary":  "Error connecting to HCP",
				"detail":   "Could not connect to HCP Terraform. Check your network.",
			},
			"type": "diagnostic",
		},
		{
			"@level":   "error",
			"@message": "Error: Network Timeout",
			"@module":  "terraform.ui",
			"diagnostic": map[string]interface{}{
				"severity": "error",
				"summary":  "Network Timeout",
				"detail":   "Connection to HCP timed out. Check your network.",
			},
			"type": "diagnostic",
		},
	}

	actual := done(t).Stdout()
	testJSONViewOutputEqualsFull(t, actual, want)
}

// These are mock error messages created solely for testing connectivity issues.
func getHCPDiags(t *testing.T) tfdiags.Diagnostics {
	t.Helper()

	var diags tfdiags.Diagnostics
	diags = diags.Append(
		tfdiags.Sourceless(
			tfdiags.Error,
			"Error connecting to HCP",
			"Could not connect to HCP Terraform. Check your network.",
		),
		&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Network Timeout",
			Detail:   "Connection to HCP timed out. Check your network.",
			Subject:  nil,
		},
	)

	return diags
}

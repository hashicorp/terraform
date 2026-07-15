// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestNewStateMigrate_LogProviderVersionSuccess(t *testing.T) {
	const verifiedChecksum = 0
	const officialProvider = 1
	const noKey = ""

	t.Run("no auth result - human view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := NewStateMigrate(arguments.ViewHuman, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		var authResult *getproviders.PackageAuthenticationResult = nil

		smView.LogProviderVersionSuccess(p, ver, authResult)

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
		smView := NewStateMigrate(arguments.ViewJSON, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		var authResult *getproviders.PackageAuthenticationResult = nil

		smView.InstalledProviderVersionInfo(p, ver, authResult)

		// Assert output
		output := done(t)
		expectedOutput := `"@message":"Installed provider version: hashicorp/test v1.2.3 (unauthenticated)"`
		if !strings.Contains(output.Stdout(), expectedOutput) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", expectedOutput, output.Stdout())
		}
	})
	t.Run("verified checksum auth result - human view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := NewStateMigrate(arguments.ViewHuman, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		authResult := getproviders.NewPackageAuthenticationResult(verifiedChecksum, noKey)

		smView.LogProviderVersionSuccess(p, ver, authResult)

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
		smView := NewStateMigrate(arguments.ViewJSON, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		authResult := getproviders.NewPackageAuthenticationResult(verifiedChecksum, noKey)

		smView.InstalledProviderVersionInfo(p, ver, authResult)

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
		smView := NewStateMigrate(arguments.ViewHuman, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		key := "key-id-123"
		authResult := getproviders.NewPackageAuthenticationResult(officialProvider, key)

		smView.LogProviderVersionSuccess(p, ver, authResult)

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
		smView := NewStateMigrate(arguments.ViewJSON, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		key := "key-id-123"
		authResult := getproviders.NewPackageAuthenticationResult(officialProvider, key)

		smView.InstalledProviderVersionInfo(p, ver, authResult)

		// Assert output - human
		output := done(t)
		expectedOutput := `"@message":"Installed provider version: hashicorp/test v1.2.3 (signed by HashiCorp)"`
		if !strings.Contains(output.Stdout(), expectedOutput) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", expectedOutput, output.Stdout())
		}
	})
}

// Assert message content
func TestNewStateMigrate_LogProviderVersionSuccessWithKeyID(t *testing.T) {
	const partnerProvider = 2

	t.Run("partner provider auth result - human view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := NewStateMigrate(arguments.ViewHuman, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		key := "key-id-123"
		authResult := getproviders.NewPackageAuthenticationResult(partnerProvider, key)

		smView.LogProviderVersionSuccessWithKeyID(p, ver, authResult, key)

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
		smView := NewStateMigrate(arguments.ViewJSON, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		key := "key-id-123"
		authResult := getproviders.NewPackageAuthenticationResult(partnerProvider, key)

		smView.InstalledProviderVersionInfoWithKeyID(p, ver, authResult, key)

		// Assert output
		output := done(t)
		expectedOutput := `"@message":"Installed provider version: hashicorp/test v1.2.3 (signed by a HashiCorp partnerkey_id: key-id-123)"`
		if !strings.Contains(output.Stdout(), expectedOutput) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", expectedOutput, output.Stdout())
		}
	})
}

func TestNewStateMigrate_Log_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := NewStateMigrate(arguments.ViewJSON, view)

	templateMessage := "This is a test log message with a parameter (%s), and both trailing whitespace and newline.    \n"
	parameter := "test_parameter"
	smView.Log(templateMessage, parameter)

	// Assert output
	output := done(t)
	expected := fmt.Sprintf(
		`"@message":"%s"`,
		strings.TrimSpace(fmt.Sprintf(templateMessage, parameter)),
	)

	if !strings.Contains(output.Stdout(), expected) {
		t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", expected, output.Stdout())
	}
}

func TestNewStateMigrate_Spacer_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := NewStateMigrate(arguments.ViewJSON, view)

	smView.Spacer()

	// Assert output
	output := done(t)

	// We cannot simply assert no output as the JSON view logs the version message on initialization
	// Splitting on \n when there's only the version log will get an array of the log and an empty string.
	// If there are more logs there'll be >2 elements.
	if x := strings.Split(output.Stdout(), "\n"); len(x) != 2 {
		t.Fatalf("expected no additional output after version message, got: %s", output.Stdout())
	}
}

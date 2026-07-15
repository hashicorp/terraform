// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"
	"testing"

	"github.com/apparentlymart/go-versions/versions"
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

		smView.LogProviderVersionSuccess(p, ver, authResult)

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

		smView.LogProviderVersionSuccess(p, ver, authResult)

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

		smView.LogProviderVersionSuccess(p, ver, authResult)

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

		smView.LogProviderVersionSuccessWithKeyID(p, ver, authResult, key)

		// Assert output
		output := done(t)
		expectedOutput := `"@message":"Installed provider version: hashicorp/test v1.2.3 (signed by a HashiCorp partnerkey_id: key-id-123)"`
		if !strings.Contains(output.Stdout(), expectedOutput) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", expectedOutput, output.Stdout())
		}
	})
}

// Assert JSON log content, including log type and additional fields
func TestNewStateMigrate_LogProviderVersionSuccess_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := NewStateMigrate(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	v := versions.MustParseVersion("1.0.0")
	officialProvider := 1
	authResult := getproviders.NewPackageAuthenticationResult(officialProvider, "key-id-123")
	smView.LogProviderVersionSuccess(p, v, authResult)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Installed provider version: hashicorp/test v1.0.0 (signed by HashiCorp)"`,
		`"@module":"terraform.ui"`,
		`"type":"installed_provider_version_info"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_ProviderAlreadyInstalled_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := NewStateMigrate(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	v := versions.MustParseVersion("1.0.0")
	smView.LogProviderVersionAlreadyInstalled(p, v)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"hashicorp/test v1.0.0: Using previously-installed provider version"`,
		`"@module":"terraform.ui"`,
		`"type":"provider_already_installed_message"`,
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
func TestNewStateMigrate_LogProviderVersionSuccessWithKeyID_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := NewStateMigrate(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	v := versions.MustParseVersion("1.0.0")
	partnerProvider := 2
	keyID := "key-id-123"
	authResult := getproviders.NewPackageAuthenticationResult(partnerProvider, keyID)
	smView.LogProviderVersionSuccessWithKeyID(p, v, authResult, keyID)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Installed provider version: hashicorp/test v1.0.0 (signed by a HashiCorp partnerkey_id: key-id-123)"`,
		`"@module":"terraform.ui"`,
		`"type":"installed_provider_version_info"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_ReusingPreviousVersion_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	initView := NewInit(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	initView.LogReusingPreviousProviderVersion(p)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"hashicorp/test: Reusing previous version from the dependency lock file"`,
		`"@module":"terraform.ui"`,
		`"type":"log"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_FindingMatchingVersion_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := NewStateMigrate(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	constraint, _ := getproviders.ParseVersionConstraints("1.0.0")
	smView.LogFindingMatchingVersion(p, constraint)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Finding matching versions for provider: hashicorp/test, version_constraint: \"1.0.0\""`,
		`"@module":"terraform.ui"`,
		`"type":"finding_matching_version_message"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_FindingLatestVersion_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := NewStateMigrate(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	smView.LogFindingLatestVersion(p)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"hashicorp/test: Finding latest version..."`,
		`"@module":"terraform.ui"`,
		`"type":"finding_latest_version_message"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_InstallingProvider_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := NewStateMigrate(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	v := versions.MustParseVersion("1.0.0")
	smView.LogInstallingProviderVersion(p, v)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Installing provider version: hashicorp/test v1.0.0..."`,
		`"@module":"terraform.ui"`,
		`"type":"installing_provider_message"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_BuiltInProviderAvailable_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := NewStateMigrate(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	smView.LogBuiltInProviderAvailable(p)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"hashicorp/test is built in to Terraform"`,
		`"@module":"terraform.ui"`,
		`"type":"built_in_provider_available_message"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_UsingProviderFromCacheDirInfo_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := NewStateMigrate(arguments.ViewJSON, view)

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	v := versions.MustParseVersion("1.0.0")
	smView.LogUsingProviderVersionFromCacheDir(p, v)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"hashicorp/test v1.0.0: Using from the shared cache directory"`,
		`"@module":"terraform.ui"`,
		`"type":"using_provider_from_cache_dir_info"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_PartnerAndCommunityProviders_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := NewStateMigrate(arguments.ViewJSON, view)

	smView.LogPartnerAndCommunityProviders()

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Partner and community providers are signed by their developers.\nIf you'd like to know more about provider signing, you can read about it here:\nhttps://developer.hashicorp.com/terraform/cli/plugins/signing"`,
		`"@module":"terraform.ui"`,
		`"type":"partner_and_community_providers_message"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_InitializingStateStoreProviderPlugin_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := NewStateMigrate(arguments.ViewJSON, view)

	storeType := "test_store"
	smView.LogInitializingStateStoreProviderPlugin(storeType)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Initializing provider plugin for state store \"test_store\"..."`,
		`"@module":"terraform.ui"`,
		`"type":"initializing_state_store_provider_plugin_message"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
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

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestNewStateMigrate_LogInstallProviderVersionComplete(t *testing.T) {
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

		smView.LogInstallProviderVersionComplete(p, ver, authResult)

		// Assert output
		output := done(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (unauthenticated)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
	})
	t.Run("verified checksum auth result - human view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := NewStateMigrate(arguments.ViewHuman, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		authResult := getproviders.NewPackageAuthenticationResult(verifiedChecksum, noKey)

		smView.LogInstallProviderVersionComplete(p, ver, authResult)

		// Assert output
		output := done(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (verified checksum)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
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

		smView.LogInstallProviderVersionComplete(p, ver, authResult)

		// Assert output
		output := done(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (signed by HashiCorp)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
	})
}

func TestNewStateMigrate_LogInstallProviderVersionCompleteWithKeyID(t *testing.T) {
	const partnerProvider = 2

	t.Run("partner provider auth result - human view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := NewStateMigrate(arguments.ViewHuman, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		key := "key-id-123"
		authResult := getproviders.NewPackageAuthenticationResult(partnerProvider, key)

		smView.LogInstallProviderVersionCompleteWithKeyID(p, ver, authResult, key)

		// Assert output - human
		output := done(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (signed by a HashiCorp partner, key ID key-id-123)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
	})
}

func TestNewStateMigrate_Spacer_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

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

func TestNewStateMigrate_LogInteractiveApproval_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	smView.LogInteractiveApproval()

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`approved by the user`, // @message - incomplete but sufficient
		`"@module":"terraform.ui"`,
		`"type":"provider_interactive_approval"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogInteractiveRejection_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	smView.LogInteractiveRejection()

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`rejected by the user`, // @message - incomplete but sufficient
		`"@module":"terraform.ui"`,
		`"type":"provider_interactive_rejection"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogAutomaticApproval_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	smView.LogAutomaticApproval()

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`approved automatically`, // @message - incomplete but sufficient
		`"@module":"terraform.ui"`,
		`"type":"provider_automatic_approval"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogDependencyLockfileCreated_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	smView.LogProviderLockfileCreated()

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`Terraform has created a lock file .terraform.lock.hcl`, // @message - incomplete but sufficient
		`"@module":"terraform.ui"`,
		`"type":"provider_lockfile_created"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogDependencyLockfileUpdated_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	smView.LogProviderLockfileUpdated()

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`Terraform has made some changes to the provider dependency selections`, // @message - incomplete but sufficient
		`"@module":"terraform.ui"`,
		`"type":"provider_lockfile_updated"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogInstallProvidersStart_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	smView.LogInstallProvidersStart()

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Installing providers..."`,
		`"@module":"terraform.ui"`,
		`"type":"provider_installation_start"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogReusingPreviousProviderVersion_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	version := getproviders.MustParseVersion("1.0.0")
	smView.LogReusingPreviousProviderVersion(p, version)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"hashicorp/test: Reusing version 1.0.0 from the dependency lock file"`,
		`"@module":"terraform.ui"`,
		`"type":"provider_query_use_previous_version"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogFindingMatchingVersion_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	constraint, _ := getproviders.ParseVersionConstraints("1.0.0")
	smView.LogFindingMatchingVersion(p, constraint)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Finding matching versions for provider: hashicorp/test, version_constraint: \"1.0.0\""`,
		`"@module":"terraform.ui"`,
		`"type":"provider_query_use_constraints"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogFindingLatestVersion_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	smView.LogFindingLatestVersion(p)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"hashicorp/test: Finding latest version..."`,
		`"@module":"terraform.ui"`,
		`"type":"provider_query_use_latest"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogProviderVersionAlreadyInstalled_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	version := getproviders.MustParseVersion("1.0.0")
	smView.LogProviderVersionAlreadyInstalled(p, version)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"hashicorp/test v1.0.0: Using previously-installed provider version"`,
		`"@module":"terraform.ui"`,
		`"type":"provider_version_already_installed"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogUsingProviderVersionFromCacheDir_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	version := getproviders.MustParseVersion("1.0.0")
	smView.LogUsingProviderVersionFromCacheDir(p, version)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"hashicorp/test v1.0.0: Using from the shared cache directory"`,
		`"@module":"terraform.ui"`,
		`"type":"provider_version_found_in_cache_dir"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogInstallProviderVersionComplete_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	v := versions.MustParseVersion("1.0.0")
	officialProvider := 1
	authResult := getproviders.NewPackageAuthenticationResult(officialProvider, "key-id-123")
	smView.LogInstallProviderVersionComplete(p, v, authResult)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Installed provider version: hashicorp/test v1.0.0 (signed by HashiCorp)"`,
		`"@module":"terraform.ui"`,
		`"type":"provider_version_installation_complete"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogInstallProviderVersionCompleteWithKeyID_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	v := versions.MustParseVersion("1.0.0")
	partnerProvider := 2
	keyID := "key-id-123"
	authResult := getproviders.NewPackageAuthenticationResult(partnerProvider, keyID)
	smView.LogInstallProviderVersionCompleteWithKeyID(p, v, authResult, keyID)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Installed provider version: hashicorp/test v1.0.0 (signed by a HashiCorp partnerkey_id: key-id-123)"`,
		`"@module":"terraform.ui"`,
		`"type":"provider_version_installation_complete"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogBuiltInProviderAvailable_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	p := addrs.MustParseProviderSourceString("hashicorp/test")
	smView.LogBuiltInProviderAvailable(p)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"hashicorp/test is built in to Terraform"`,
		`"@module":"terraform.ui"`,
		`"type":"built_in_provider_available"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestProvidersLockHuman_LogProviderLockfileUpdated(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewProvidersLock(arguments.ViewHuman, view)

	v.LogProviderLockfileUpdated()

	output := done(t)

	// Assert contents
	// This must be done separately for stdout and stderr due to
	// the interleaving of output caused in tests by (TestOutput).All()
	gotStdout := strings.TrimSpace(output.Stdout())
	wantStdout := "Terraform has updated the lock file"
	if !strings.Contains(gotStdout, wantStdout) {
		t.Fatalf("expected stdout to contain %q, got %q", wantStdout, gotStdout)
	}
	gotStderr := strings.TrimSpace(output.Stderr())
	wantStderr := ""
	if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
}

func TestProvidersLockHuman_LogLockFileValidated(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewProvidersLock(arguments.ViewHuman, view)

	v.LogLockFileValidated()

	output := done(t)

	// Assert contents
	// This must be done separately for stdout and stderr due to
	// the interleaving of output caused in tests by (TestOutput).All()
	gotStdout := strings.TrimSpace(output.Stdout())
	wantStdout := "Terraform has validated the lock file"
	if !strings.Contains(gotStdout, wantStdout) {
		t.Fatalf("expected stdout to contain %q, got %q", wantStdout, gotStdout)
	}
	gotStderr := strings.TrimSpace(output.Stderr())
	wantStderr := ""
	if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
}

func TestProvidersLockHuman_LogInstallProviderVersionStart(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewProvidersLock(arguments.ViewHuman, view)

	providerAddr := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/aws")
	version := getproviders.MustParseVersion("4.0.0")
	platform, err := getproviders.ParsePlatform("linux_amd64")
	if err != nil {
		t.Fatalf("failed to parse platform: %s", err)
	}

	v.LogInstallProviderVersionStart(providerAddr, version, platform)

	output := done(t)

	// Assert contents
	// This must be done separately for stdout and stderr due to
	// the interleaving of output caused in tests by (TestOutput).All()
	gotStdout := strings.TrimSpace(output.Stdout())
	wantStdout := "Fetching hashicorp/aws 4.0.0 for linux_amd64"
	if !strings.Contains(gotStdout, wantStdout) {
		t.Fatalf("expected stdout to contain %q, got %q", wantStdout, gotStdout)
	}
	gotStderr := strings.TrimSpace(output.Stderr())
	wantStderr := ""
	if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
}

func TestProvidersLockHuman_LogInstallProviderVersionComplete(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewProvidersLock(arguments.ViewHuman, view)

	providerAddr := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/aws")
	version := getproviders.MustParseVersion("4.0.0")
	platform, err := getproviders.ParsePlatform("linux_amd64")
	if err != nil {
		t.Fatalf("failed to parse platform: %s", err)
	}
	auth := getproviders.NewPackageAuthenticationResult(2, "third-party-key-id")

	v.LogInstallProviderVersionComplete(providerAddr, version, platform, auth)

	output := done(t)

	// Assert contents
	// This must be done separately for stdout and stderr due to
	// the interleaving of output caused in tests by (TestOutput).All()
	gotStdout := strings.TrimSpace(output.Stdout())
	wantStdout := "Retrieved hashicorp/aws 4.0.0 for linux_amd64 (signed by a HashiCorp partner)"
	if !strings.Contains(gotStdout, wantStdout) {
		t.Fatalf("expected stdout to contain %q, got %q", wantStdout, gotStdout)
	}
	gotStderr := strings.TrimSpace(output.Stderr())
	wantStderr := ""
	if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
}

func TestProvidersLockHuman_LogInstallProviderVersionCompleteWithKeyID(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewProvidersLock(arguments.ViewHuman, view)

	providerAddr := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/aws")
	version := getproviders.MustParseVersion("4.0.0")
	platform, err := getproviders.ParsePlatform("linux_amd64")
	if err != nil {
		t.Fatalf("failed to parse platform: %s", err)
	}
	auth := getproviders.NewPackageAuthenticationResult(2, "third-party-key-id")

	v.LogInstallProviderVersionCompleteWithKeyID(providerAddr, version, platform, auth, "third-party-key-id")

	output := done(t)

	// Assert contents
	// This must be done separately for stdout and stderr due to
	// the interleaving of output caused in tests by (TestOutput).All()
	gotStdout := strings.TrimSpace(output.Stdout())
	wantStdout := "Retrieved hashicorp/aws 4.0.0 for linux_amd64 (signed by a HashiCorp partner, key ID third-party-key-id)"
	if !strings.Contains(gotStdout, wantStdout) {
		t.Fatalf("expected stdout to contain %q, got %q", wantStdout, gotStdout)
	}
	gotStderr := strings.TrimSpace(output.Stderr())
	wantStderr := ""
	if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
}

func TestProvidersLockHuman_LogProviderAddedToLockFile(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewProvidersLock(arguments.ViewHuman, view)

	providerAddr := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/aws")
	platform, err := getproviders.ParsePlatform("linux_amd64")
	if err != nil {
		t.Fatalf("failed to parse platform: %s", err)
	}

	v.LogProviderAddedToLockFile(providerAddr, platform)

	output := done(t)

	// Assert contents
	// This must be done separately for stdout and stderr due to
	// the interleaving of output caused in tests by (TestOutput).All()
	gotStdout := strings.TrimSpace(output.Stdout())
	wantStdout := "Obtained hashicorp/aws checksums for linux_amd64; This was a new provider"
	if !strings.Contains(gotStdout, wantStdout) {
		t.Fatalf("expected stdout to contain %q, got %q", wantStdout, gotStdout)
	}
	gotStderr := strings.TrimSpace(output.Stderr())
	wantStderr := ""
	if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
}

func TestProvidersLockHuman_LogProviderChecksumsUpdatedInLockFile(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewProvidersLock(arguments.ViewHuman, view)

	providerAddr := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/aws")
	platform, err := getproviders.ParsePlatform("linux_amd64")
	if err != nil {
		t.Fatalf("failed to parse platform: %s", err)
	}

	v.LogProviderChecksumsUpdatedInLockFile(providerAddr, platform)

	output := done(t)

	// Assert contents
	// This must be done separately for stdout and stderr due to
	// the interleaving of output caused in tests by (TestOutput).All()
	gotStdout := strings.TrimSpace(output.Stdout())
	wantStdout := "Obtained hashicorp/aws checksums for linux_amd64; Additional checksums"
	if !strings.Contains(gotStdout, wantStdout) {
		t.Fatalf("expected stdout to contain %q, got %q", wantStdout, gotStdout)
	}
	gotStderr := strings.TrimSpace(output.Stderr())
	wantStderr := ""
	if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
}

func TestProvidersLockHuman_LogProviderUnchangedInLockFile(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewProvidersLock(arguments.ViewHuman, view)

	providerAddr := addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/aws")
	platform, err := getproviders.ParsePlatform("linux_amd64")
	if err != nil {
		t.Fatalf("failed to parse platform: %s", err)
	}

	v.LogProviderUnchangedInLockFile(providerAddr, platform)

	output := done(t)

	// Assert contents
	// This must be done separately for stdout and stderr due to
	// the interleaving of output caused in tests by (TestOutput).All()
	gotStdout := strings.TrimSpace(output.Stdout())
	wantStdout := "Obtained hashicorp/aws checksums for linux_amd64; All checksums for this platform were already tracked in the lock file"
	if !strings.Contains(gotStdout, wantStdout) {
		t.Fatalf("expected stdout to contain %q, got %q", wantStdout, gotStdout)
	}
	gotStderr := strings.TrimSpace(output.Stderr())
	wantStderr := ""
	if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
}

func TestProvidersLockHuman_Diagnostics(t *testing.T) {
	testCases := map[string]struct {
		diags      tfdiags.Diagnostics
		wantStdout string
		wantStderr string
	}{
		"error": {
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Example error",
					"This is an example error message.",
				),
			},
			"",
			"Error: Example error\n\nThis is an example error message.\n\n",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewProvidersLock(arguments.ViewHuman, view)

			v.Diagnostics(tc.diags)

			output := done(t)

			// Assert contents
			// This must be done separately for stdout and stderr due to
			// the interleaving of output caused in tests by (TestOutput).All()
			gotStdout := strings.TrimSpace(output.Stdout())
			wantStdout := strings.TrimSpace(tc.wantStdout)
			if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
			gotStderr := strings.TrimSpace(output.Stderr())
			wantStderr := strings.TrimSpace(tc.wantStderr)
			if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
		})
	}
}

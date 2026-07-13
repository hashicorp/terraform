// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestNewStateMigrate_InstalledProviderVersionInfo(t *testing.T) {
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

		smView.InstalledProviderVersionInfo(p, ver, authResult)

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

		smView.InstalledProviderVersionInfo(p, ver, authResult)

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

		smView.InstalledProviderVersionInfo(p, ver, authResult)

		// Assert output
		output := done(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (signed by HashiCorp)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
	})
}

func TestNewStateMigrate_InstalledProviderVersionInfoWithKeyID(t *testing.T) {
	const partnerProvider = 2

	t.Run("partner provider auth result - human view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := NewStateMigrate(arguments.ViewHuman, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		key := "key-id-123"
		authResult := getproviders.NewPackageAuthenticationResult(partnerProvider, key)

		smView.InstalledProviderVersionInfoWithKeyID(p, ver, authResult, key)

		// Assert output - human
		output := done(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (signed by a HashiCorp partner, key ID key-id-123)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
	})
}

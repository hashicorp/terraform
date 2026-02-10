// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestProvidersLockHuman_Fetching(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewProvidersLock(NewView(streams))
	v.(*ProvidersLockHuman).view.Configure(&arguments.View{NoColor: true})

	provider := addrs.NewDefaultProvider("test")
	version := getproviders.MustParseVersion("1.0.0")
	platform := getproviders.Platform{OS: "linux", Arch: "amd64"}

	v.Fetching(provider, version, platform)

	got := done(t).Stdout()
	want := "- Fetching hashicorp/test 1.0.0 for linux_amd64...\n"
	if got != want {
		t.Fatalf("wrong output\n got: %q\nwant: %q", got, want)
	}
}

func TestProvidersLockHuman_FetchSuccess(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewProvidersLock(NewView(streams))
	v.(*ProvidersLockHuman).view.Configure(&arguments.View{NoColor: true})

	provider := addrs.NewDefaultProvider("test")
	version := getproviders.MustParseVersion("1.0.0")
	platform := getproviders.Platform{OS: "linux", Arch: "amd64"}

	v.FetchSuccess(provider, version, platform, "signed", "")

	got := done(t).Stdout()
	want := "- Retrieved hashicorp/test 1.0.0 for linux_amd64 (signed)\n"
	if got != want {
		t.Fatalf("wrong output\n got: %q\nwant: %q", got, want)
	}
}

func TestProvidersLockHuman_FetchSuccess_withKeyID(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewProvidersLock(NewView(streams))
	v.(*ProvidersLockHuman).view.Configure(&arguments.View{NoColor: true})

	provider := addrs.NewDefaultProvider("test")
	version := getproviders.MustParseVersion("1.0.0")
	platform := getproviders.Platform{OS: "linux", Arch: "amd64"}

	v.FetchSuccess(provider, version, platform, "signed", "ABC123")

	got := done(t).Stdout()
	if !strings.Contains(got, "ABC123") {
		t.Fatalf("expected output to contain key ID, got: %q", got)
	}
}

func TestProvidersLockHuman_NewProvider(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewProvidersLock(NewView(streams))

	provider := addrs.NewDefaultProvider("test")
	platform := getproviders.Platform{OS: "linux", Arch: "amd64"}

	v.NewProvider(provider, platform)

	got := done(t).Stdout()
	if !strings.Contains(got, "new provider") {
		t.Fatalf("expected output to mention new provider, got: %q", got)
	}
}

func TestProvidersLockHuman_NewHashes(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewProvidersLock(NewView(streams))

	provider := addrs.NewDefaultProvider("test")
	platform := getproviders.Platform{OS: "linux", Arch: "amd64"}

	v.NewHashes(provider, platform)

	got := done(t).Stdout()
	if !strings.Contains(got, "Additional checksums") {
		t.Fatalf("expected output to mention additional checksums, got: %q", got)
	}
}

func TestProvidersLockHuman_ExistingHashes(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewProvidersLock(NewView(streams))

	provider := addrs.NewDefaultProvider("test")
	platform := getproviders.Platform{OS: "linux", Arch: "amd64"}

	v.ExistingHashes(provider, platform)

	got := done(t).Stdout()
	if !strings.Contains(got, "already tracked") {
		t.Fatalf("expected output to mention already tracked, got: %q", got)
	}
}

func TestProvidersLockHuman_Success_withChanges(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewProvidersLock(NewView(streams))
	v.(*ProvidersLockHuman).view.Configure(&arguments.View{NoColor: true})

	v.Success(true)

	got := done(t).Stdout()
	if !strings.Contains(got, "updated the lock file") {
		t.Fatalf("expected output to mention updated lock file, got: %q", got)
	}
	if !strings.Contains(got, "Review the changes") {
		t.Fatalf("expected output to mention reviewing changes, got: %q", got)
	}
}

func TestProvidersLockHuman_Success_noChanges(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewProvidersLock(NewView(streams))
	v.(*ProvidersLockHuman).view.Configure(&arguments.View{NoColor: true})

	v.Success(false)

	got := done(t).Stdout()
	if !strings.Contains(got, "no need for changes") {
		t.Fatalf("expected output to mention no need for changes, got: %q", got)
	}
}

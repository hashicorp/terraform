// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProvidersLock is the view interface for the providers lock command.
type ProvidersLock interface {
	// Fetching announces that a provider package is being fetched.
	Fetching(provider addrs.Provider, version getproviders.Version, platform getproviders.Platform)

	// FetchSuccess announces that a provider package was fetched successfully.
	FetchSuccess(provider addrs.Provider, version getproviders.Version, platform getproviders.Platform, auth string, keyID string)

	// NewProvider announces that checksums for a new provider have been added to the lock file.
	NewProvider(provider addrs.Provider, platform getproviders.Platform)

	// NewHashes announces that additional checksums have been added for an existing provider.
	NewHashes(provider addrs.Provider, platform getproviders.Platform)

	// ExistingHashes announces that all checksums were already tracked.
	ExistingHashes(provider addrs.Provider, platform getproviders.Platform)

	// Success announces a successful completion, with whether any changes were made.
	Success(madeChanges bool)

	// Diagnostics renders diagnostics.
	Diagnostics(diags tfdiags.Diagnostics)

	// HelpPrompt directs the user to the help output.
	HelpPrompt()
}

// NewProvidersLock returns an initialized ProvidersLock implementation.
func NewProvidersLock(view *View) ProvidersLock {
	return &ProvidersLockHuman{view: view}
}

// ProvidersLockHuman is the human-readable implementation of the ProvidersLock view.
type ProvidersLockHuman struct {
	view *View
}

var _ ProvidersLock = (*ProvidersLockHuman)(nil)

func (v *ProvidersLockHuman) Fetching(provider addrs.Provider, version getproviders.Version, platform getproviders.Platform) {
	v.view.streams.Println(fmt.Sprintf("- Fetching %s %s for %s...", provider.ForDisplay(), version, platform))
}

func (v *ProvidersLockHuman) FetchSuccess(provider addrs.Provider, version getproviders.Version, platform getproviders.Platform, auth string, keyID string) {
	if keyID != "" {
		keyID = v.view.colorize.Color(fmt.Sprintf(", key ID [reset][bold]%s[reset]", keyID))
	}
	v.view.streams.Println(fmt.Sprintf("- Retrieved %s %s for %s (%s%s)", provider.ForDisplay(), version, platform, auth, keyID))
}

func (v *ProvidersLockHuman) NewProvider(provider addrs.Provider, platform getproviders.Platform) {
	v.view.streams.Println(fmt.Sprintf(
		"- Obtained %s checksums for %s; This was a new provider and the checksums for this platform are now tracked in the lock file",
		provider.ForDisplay(),
		platform))
}

func (v *ProvidersLockHuman) NewHashes(provider addrs.Provider, platform getproviders.Platform) {
	v.view.streams.Println(fmt.Sprintf(
		"- Obtained %s checksums for %s; Additional checksums for this platform are now tracked in the lock file",
		provider.ForDisplay(),
		platform))
}

func (v *ProvidersLockHuman) ExistingHashes(provider addrs.Provider, platform getproviders.Platform) {
	v.view.streams.Println(fmt.Sprintf(
		"- Obtained %s checksums for %s; All checksums for this platform were already tracked in the lock file",
		provider.ForDisplay(),
		platform))
}

func (v *ProvidersLockHuman) Success(madeChanges bool) {
	if madeChanges {
		v.view.streams.Println(v.view.colorize.Color("\n[bold][green]Success![reset] [bold]Terraform has updated the lock file.[reset]"))
		v.view.streams.Println("\nReview the changes in .terraform.lock.hcl and then commit to your\nversion control system to retain the new checksums.")
	} else {
		v.view.streams.Println(v.view.colorize.Color("\n[bold][green]Success![reset] [bold]Terraform has validated the lock file and found no need for changes.[reset]"))
	}
}

func (v *ProvidersLockHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *ProvidersLockHuman) HelpPrompt() {
	v.view.HelpPrompt("providers lock")
}

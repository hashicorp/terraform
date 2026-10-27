// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProvidersLock is the view used for the `providers lock` command.
type ProvidersLock interface {
	// TODO: Refactor to use ProviderLockingLogger. This would require distinguishing between lockfile creation and update
	// in the `providers lock` command.
	LogProviderLockfileUpdated()

	// Log that the lock file was validated and no changes were necessary.
	LogLockFileValidated()

	// Log that a provider was added to the lock file.
	LogProviderAddedToLockFile(providerAddr addrs.Provider, platform getproviders.Platform)

	// Log that checksums for a provider were added to the lock file.
	LogProviderChecksumsUpdatedInLockFile(providerAddr addrs.Provider, platform getproviders.Platform)

	// Log that a lock file entry for a provider was validated, i.e nothing changed.
	LogProviderUnchangedInLockFile(providerAddr addrs.Provider, platform getproviders.Platform)

	Diagnostics(diags tfdiags.Diagnostics)

	/*
		FIXME: The methods below resemble a subset of the ProviderInstallationLogger interface; should they be combined?
		This would require splitting up the interface, but also accounting for how auth info is rendered for `provider lock`
		output but not in `init` output.
	*/

	// LogInstallProviderVersionStart indicates that a provider is being installed (from a remote location)
	LogInstallProviderVersionStart(providerAddr addrs.Provider, version getproviders.Version, platform getproviders.Platform)

	// LogInstallProviderVersionCompleteWithKeyID indicates that a provider installation has completed with a specific key ID
	LogInstallProviderVersionCompleteWithKeyID(providerAddr addrs.Provider, version getproviders.Version, platform getproviders.Platform, auth *getproviders.PackageAuthenticationResult, keyID string)

	// LogInstallProviderVersionComplete indicates that a provider installation has completed
	LogInstallProviderVersionComplete(providerAddr addrs.Provider, version getproviders.Version, platform getproviders.Platform, auth *getproviders.PackageAuthenticationResult)
}

// Note: if JSON output is added in future, `providers lock` should produce SRO-style JSON output.
func NewProvidersLock(viewType arguments.ViewType, view *View) ProvidersLock {
	switch viewType {
	case arguments.ViewHuman:
		return &ProvidersLockHuman{view: view}
	default:
		panic("unsupported view type: " + string(viewType))
	}
}

// ProvidersLockHuman renders module installation progress and diagnostics for direct consumption.
type ProvidersLockHuman struct {
	view *View
}

var _ ProvidersLock = (*ProvidersLockHuman)(nil)

func (v *ProvidersLockHuman) LogProviderAddedToLockFile(providerAddr addrs.Provider, platform getproviders.Platform) {
	v.view.streams.Println(
		fmt.Sprintf(
			"- Obtained %s checksums for %s; This was a new provider and the checksums for this platform are now tracked in the lock file",
			providerAddr.ForDisplay(),
			platform,
		),
	)
}

func (v *ProvidersLockHuman) LogProviderChecksumsUpdatedInLockFile(providerAddr addrs.Provider, platform getproviders.Platform) {
	v.view.streams.Println(
		fmt.Sprintf(
			"- Obtained %s checksums for %s; Additional checksums for this platform are now tracked in the lock file",
			providerAddr.ForDisplay(),
			platform,
		),
	)
}

func (v *ProvidersLockHuman) LogProviderUnchangedInLockFile(providerAddr addrs.Provider, platform getproviders.Platform) {
	v.view.streams.Println(
		fmt.Sprintf(
			"- Obtained %s checksums for %s; All checksums for this platform were already tracked in the lock file",
			providerAddr.ForDisplay(),
			platform),
	)
}

func (v *ProvidersLockHuman) LogProviderLockfileUpdated() {
	v.view.streams.Print(v.view.colorize.Color("\n[bold][green]Success![reset] [bold]Terraform has updated the lock file.[reset]\n"))
	v.view.streams.Print("\nReview the changes in .terraform.lock.hcl and then commit to your\nversion control system to retain the new checksums.\n")
}

func (v *ProvidersLockHuman) LogLockFileValidated() {
	v.view.streams.Print(v.view.colorize.Color("\n[bold][green]Success![reset] [bold]Terraform has validated the lock file and found no need for changes.[reset]\n"))
}

// Distinct from LogInstallProviderVersionStart on the ProviderInstallationLogger interface
// TODO: Address discrepancy between `init` and `providers lock` logging.
func (v *ProvidersLockHuman) LogInstallProviderVersionStart(providerAddr addrs.Provider, version getproviders.Version, platform getproviders.Platform) {
	v.view.streams.Println(fmt.Sprintf("- Fetching %s %s for %s...", providerAddr.ForDisplay(), version, platform))
}

// Distinct from LogInstallProviderVersionCompleteWithKeyID on the ProviderInstallationLogger interface
// TODO: Address discrepancy between `init` and `providers lock` logging.
func (v *ProvidersLockHuman) LogInstallProviderVersionCompleteWithKeyID(providerAddr addrs.Provider, version getproviders.Version, platform getproviders.Platform, auth *getproviders.PackageAuthenticationResult, keyID string) {
	v.view.streams.Println(
		v.view.colorize.Color(
			fmt.Sprintf("- Retrieved %s %s for %s (%s, key ID [reset][bold]%s[reset])", providerAddr.ForDisplay(), version, platform, auth, keyID),
		),
	)
}

// Distinct from LogInstallProviderVersionComplete on the ProviderInstallationLogger interface
// TODO: Address discrepancy between `init` and `providers lock` logging.
func (v *ProvidersLockHuman) LogInstallProviderVersionComplete(providerAddr addrs.Provider, version getproviders.Version, platform getproviders.Platform, auth *getproviders.PackageAuthenticationResult) {
	v.view.streams.Println(
		v.view.colorize.Color(
			fmt.Sprintf("- Retrieved %s %s for %s (%s)", providerAddr.ForDisplay(), version, platform, auth),
		),
	)
}

func (v *ProvidersLockHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

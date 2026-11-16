// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProvidersMirror is the view used for the `providers mirror` command.
type ProvidersMirror interface {
	// This block of methods is a subset of the ProviderInstallationLogger interface
	// TODO: reconcile this, maybe by splitting that interface up.
	LogBuiltInProviderAvailable(providerAddr addrs.Provider)
	LogReusingPreviousProviderVersion(ver versions.Version)                                             // different signature vs ProviderInstallationLogger interface
	LogFindingMatchingVersion(ver versions.Version, versionConstraints providerreqs.VersionConstraints) // different signature vs ProviderInstallationLogger interface
	LogFindingLatestVersion(ver versions.Version)                                                       // different signature vs ProviderInstallationLogger interface
	LogInstallProviderVersionStart(platform getproviders.Platform)                                      // different signature vs ProviderInstallationLogger interface
	LogInstallProviderVersionComplete(auth *getproviders.PackageAuthenticationResult)                   // different signature vs ProviderInstallationLogger interface

	MirrorProviderStart(providerAddr addrs.Provider)

	Diagnostics(diags tfdiags.Diagnostics)
}

// Note: if JSON output is added in future, `providers lock` should produce SRO-style JSON output.
func NewProvidersMirror(viewType arguments.ViewType, view *View) ProvidersMirror {
	switch viewType {
	case arguments.ViewHuman:
		return &ProvidersMirrorHuman{view: view}
	default:
		panic("unsupported view type: " + string(viewType))
	}
}

// ProvidersMirrorHuman renders module installation progress and diagnostics for direct consumption.
type ProvidersMirrorHuman struct {
	view *View
}

var _ ProvidersMirror = (*ProvidersMirrorHuman)(nil)

func (v *ProvidersMirrorHuman) LogBuiltInProviderAvailable(providerAddr addrs.Provider) {
	v.view.streams.Println(
		fmt.Sprintf("- Skipping %s because it is built in to Terraform CLI", providerAddr.ForDisplay()),
	)
}

func (v *ProvidersMirrorHuman) MirrorProviderStart(providerAddr addrs.Provider) {
	v.view.streams.Println(
		fmt.Sprintf("- Mirroring %s...", providerAddr.ForDisplay()),
	)
}

func (v *ProvidersMirrorHuman) LogReusingPreviousProviderVersion(ver versions.Version) {
	v.view.streams.Println(
		fmt.Sprintf("  - Selected v%s to match dependency lock file", ver.String()),
	)
}

func (v *ProvidersMirrorHuman) LogFindingMatchingVersion(ver versions.Version, versionConstraints providerreqs.VersionConstraints) {
	v.view.streams.Println(
		fmt.Sprintf("  - Selected v%s to meet constraints %s", ver.String(), versionConstraints),
	)
}

func (v *ProvidersMirrorHuman) LogFindingLatestVersion(ver versions.Version) {
	v.view.streams.Println(
		fmt.Sprintf("  - Selected v%s with no constraints", ver.String()),
	)
}

func (v *ProvidersMirrorHuman) LogInstallProviderVersionStart(platform getproviders.Platform) {
	v.view.streams.Println(
		fmt.Sprintf("  - Downloading package for %s...", platform.String()),
	)
}

func (v *ProvidersMirrorHuman) LogInstallProviderVersionComplete(auth *getproviders.PackageAuthenticationResult) {
	v.view.streams.Println(
		fmt.Sprintf("  - Package authenticated: %s", auth.String()),
	)
}

func (v *ProvidersMirrorHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

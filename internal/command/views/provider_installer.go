// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
)

// ProviderInstaller is an interface that describes the methods required by a view that's used
// with provider installation methods.
//
// The `Output` method is a constraint from the Init view interface, which will be refactored away soon.
type ProviderInstaller interface {
	Output(messageCode InitMessageCode, params ...any)

	// LogProviderVersionSuccess describes a successfully installed provider along with its version
	LogProviderVersionSuccess(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult)

	// LogProviderVersionSuccessWithKeyID describes a successfully installed provider along with its version and the key ID used to verify the provider's authenticity
	LogProviderVersionSuccessWithKeyID(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult, keyID string)

	// LogProviderVersionAlreadyInstalled indicates a provider that is already installed during installation
	LogProviderVersionAlreadyInstalled(providerAddr addrs.Provider, version getproviders.Version)

	// LogReusingPreviousProviderVersion indicates a provider is locked to a specific version during installation
	LogReusingPreviousProviderVersion(providerAddr addrs.Provider, version getproviders.Version)

	// LogFindingMatchingVersion indicates that Terraform is looking for a provider version that matches the constraint during installation.
	LogFindingMatchingVersion(providerAddr addrs.Provider, versionConstraints getproviders.VersionConstraints)

	// FindingLatestVersion indicates that Terraform is looking for the latest version of a provider during installation (no constraint nor prior lock was supplied)
	LogFindingLatestVersion(providerAddr addrs.Provider)

	// LogInstallingProviderVersion indicates that a provider is being installed (from a remote location)
	LogInstallingProviderVersion(providerAddr addrs.Provider, version getproviders.Version)

	// LogBuiltInProviderAvailable indicates a built-in provider is available in the current Terraform core binary and is in use during installation
	LogBuiltInProviderAvailable(providerAddr addrs.Provider)

	// LogUsingProviderVersionFromCacheDir indicates that a provider is being linked from a system-wide cache, instead of being downloaded from an external source.
	LogUsingProviderVersionFromCacheDir(providerAddr addrs.Provider, version getproviders.Version)

	// Log that a provider successfully fetched in this operation is maintained by third-parties and describe how these are signed
	LogPartnerAndCommunityProviders()

	// LogInitializingStateStoreProviderPlugin indicates progress during installation of a state store provider plugin
	LogInitializingStateStoreProviderPlugin(providerAddr addrs.Provider, cons getproviders.VersionConstraints, storeType string)

	prepareMessage(messageCode InitMessageCode, params ...any) string

	Spacer // output from provider installation is spaced out from following human-readable output log lines
}

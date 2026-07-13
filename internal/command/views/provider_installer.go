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

	// InitializingStateStoreProviderPlugin indicates progress during installation of a state store provider plugin.
	InitializingStateStoreProviderPlugin(storeType string)

	// FindingMatchingVersion indicates that Terraform is looking for a provider version that matches the constraint during installation
	FindingMatchingVersion(providerAddr addrs.Provider, versionConstraints getproviders.VersionConstraints)

	// FindingLatestVersion indicates that Terraform is looking for the latest version of a provider during installation (no constraint was supplied)
	FindingLatestVersion(providerAddr addrs.Provider)

	// InstallingProvider indicates that a provider is being installed (from a remote location)
	InstallingProvider(providerAddr addrs.Provider, version getproviders.Version)

	// ProviderAlreadyInstalled indicates a provider that is already installed during installation
	ProviderAlreadyInstalled(providerAddr addrs.Provider, version getproviders.Version)

	// UsingProviderFromCacheDirInfo indicates that a provider is being linked from a system-wide cache
	UsingProviderFromCacheDirInfo(providerAddr addrs.Provider, version getproviders.Version)

	// BuiltInProviderAvailable indicates a built-in provider in use during installation
	BuiltInProviderAvailable(providerAddr addrs.Provider)

	// ReusingPreviousVersion indicates a provider which is locked to a specific version during installation
	ReusingPreviousVersion(providerAddr addrs.Provider)

	// InstalledProviderVersionInfo describes a successfully installed provider along with its version
	InstalledProviderVersionInfo(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult)

	// InstalledProviderVersionInfoWithKeyID describes a successfully installed provider along with its version and the key ID used to verify the provider's authenticity
	InstalledProviderVersionInfoWithKeyID(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult, keyID string)

	// PartnerAndCommunityProviders is a message concerning partner and community providers and how these are signed
	PartnerAndCommunityProviders()

	// LockfileCreated indicates that a dependency lock file was created during installation
	LockfileCreated()

	// LockfileUpdated indicates that a dependency lock file was updated during installation
	LockfileUpdated()

	prepareMessage(messageCode InitMessageCode, params ...any) string

	Spacer // output from provider installation is spaced out from following human-readable output log lines
}

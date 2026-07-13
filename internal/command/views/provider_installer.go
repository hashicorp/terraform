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
// The method names here are constrained by the Init view interface, which is coupled to the
// provider installation process. In a future major version of Terraform this could be improved.
// See: https://github.com/hashicorp/terraform/issues/38763
type ProviderInstaller interface {
	LogInitMessage(messageCode InitMessageCode, params ...any)
	Output(messageCode InitMessageCode, params ...any)

	// Log details about a successfully fetched provider package.
	LogProviderVersionSuccess(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult)

	// Log details about a successfully fetched provider package, including details about the key used to sign it.
	LogProviderVersionSuccessWithKeyID(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult, keyID string)

	// Log details about a successfully fetched provider package
	LogProviderAlreadyInstalled(providerAddr addrs.Provider, version getproviders.Version)

	// Log details about a provider being controlled by a pre-existing lock in a dependency lock file.
	LogReusingPreviousProviderVersion(providerAddr addrs.Provider)

	// Log details about a provider installation being controlled by a version constraint.
	LogFindingMatchingVersion(providerAddr addrs.Provider, versionConstraints getproviders.VersionConstraints)

	// Log details about a provider installation not being controlled by a version constraint nor a dependency lock file; latest available version is used.
	LogFindingLatestVersion(providerAddr addrs.Provider)

	// Log details about a provider installation process that's starting.
	LogInstallingProvider(providerAddr addrs.Provider, version getproviders.Version)

	// Log that the built-in provider is available in the current Terraform core binary
	LogBuiltInProviderAvailable(providerAddr addrs.Provider)

	// Log that the provider version in use is being used from a local cache instead of being downloaded from an external source.
	LogUsingProviderFromCacheDir(providerAddr addrs.Provider, version getproviders.Version)

	prepareMessage(messageCode InitMessageCode, params ...any) string

	Spacer // output from provider installation is spaced out from following human-readable output log lines
}

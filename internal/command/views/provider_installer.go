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

	InstalledProviderVersionInfo(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult)
	InstalledProviderVersionInfoWithKeyID(providerAddr addrs.Provider, version getproviders.Version, auth *getproviders.PackageAuthenticationResult, keyID string)

	prepareMessage(messageCode InitMessageCode, params ...any) string

	Spacer // output from provider installation is spaced out from following human-readable output log lines
}

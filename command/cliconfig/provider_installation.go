package cliconfig

import (
	"fmt"
)

// ProviderInstallationLocation is an interface type representing the
// different installation location types. The concrete implementations of
// this interface are:
//
//     ProviderInstallationDirect:                install from the provider's origin registry
//     ProviderInstallationFilesystemMirror(dir): install from a local filesystem mirror
//     ProviderInstallationNetworkMirror(host):   install from a network mirror
type ProviderInstallationLocation interface {
	providerInstallationLocation()
}

type providerInstallationDirect [0]byte

func (i providerInstallationDirect) providerInstallationLocation() {}

// ProviderInstallationDirect is a ProviderInstallationSourceLocation
// representing installation from a provider's origin registry.
var ProviderInstallationDirect ProviderInstallationLocation = providerInstallationDirect{}

func (i providerInstallationDirect) GoString() string {
	return "cliconfig.ProviderInstallationDirect"
}

// ProviderInstallationFilesystemMirror is a ProviderInstallationSourceLocation
// representing installation from a particular local filesystem mirror. The
// string value is the filesystem path to the mirror directory.
type ProviderInstallationFilesystemMirror string

func (i ProviderInstallationFilesystemMirror) providerInstallationLocation() {}

func (i ProviderInstallationFilesystemMirror) GoString() string {
	return fmt.Sprintf("cliconfig.ProviderInstallationFilesystemMirror(%q)", i)
}

// ProviderInstallationNetworkMirror is a ProviderInstallationSourceLocation
// representing installation from a particular local network mirror. The
// string value is the HTTP base URL exactly as written in the configuration,
// without any normalization.
type ProviderInstallationNetworkMirror string

func (i ProviderInstallationNetworkMirror) providerInstallationLocation() {}

func (i ProviderInstallationNetworkMirror) GoString() string {
	return fmt.Sprintf("cliconfig.ProviderInstallationNetworkMirror(%q)", i)
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// The version package provides a location to set the release versions for all
// packages to consume, without creating import cycles.
//
// This package should not import any other terraform packages.
package version

import (
	_ "embed"
	"fmt"
	"strings"

	version "github.com/hashicorp/go-version"
)

// rawVersion is the current version as a string, as read from the VERSION
// file. This must be a valid semantic version.
//
//go:embed VERSION
var rawVersion string

// dev determines whether the -dev prerelease marker will
// be included in version info. It is expected to be set to "no" using
// linker flags when building binaries for release.
var dev string = "yes"

// The main version number that is being run at the moment, populated from the raw version.
var Version string

// A pre-release marker for the version, populated using a combination of the raw version
// and the dev flag.
var Prerelease string

// SemVer is an instance of version.Version representing the main version
// without any prerelease information.
var SemVer *version.Version

func init() {
	semVerFull := version.Must(version.NewVersion(strings.TrimSpace(rawVersion)))
	SemVer = semVerFull.Core()
	Version = SemVer.String()

	if dev == "no" {
		Prerelease = semVerFull.Prerelease()
	} else {
		Prerelease = "dev"
	}
}

// Header is the header name used to send the current terraform version
// in http requests.
const Header = "Terraform-Version"

// String returns the complete version string, including prerelease
func String() string {
	if Prerelease != "" {
		return fmt.Sprintf("%s-%s", Version, Prerelease)
	}
	return Version
}

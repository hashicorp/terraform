// The version package provides a location to set the release versions for all
// packages to consume, without creating import cycles.
//
// This package should not import any other terraform packages.
package version

import (
	"fmt"

	version "github.com/hashicorp/go-version"
)

// The main version number that is being run at the moment.
const Version = "0.11.8"

// A pre-release marker for the version. If this is "" (empty string)
// then it means that it is a final release. Otherwise, this is a pre-release
// such as "dev" (in development), "beta", "rc1", etc.
var Prerelease = "dev"

// SemVer is an instance of version.Version. This has the secondary
// benefit of verifying during tests and init time that our version is a
// proper semantic version, which should always be the case.
var SemVer = version.Must(version.NewVersion(Version))

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

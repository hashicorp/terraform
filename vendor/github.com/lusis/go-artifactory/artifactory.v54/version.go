package artifactory

import "github.com/coreos/go-semver/semver"

var (
	// VersionMajor is for an API incompatible changes
	VersionMajor int64 = 5
	// VersionMinor is for functionality in a backwards-compatible manner
	VersionMinor int64 = 4
	// VersionPatch is for backwards-compatible bug fixes
	VersionPatch int64
)

// Version represents the minimum version of the Artifactory API this library supports
var Version = semver.Version{
	Major: VersionMajor,
	Minor: VersionMinor,
	Patch: VersionPatch,
}

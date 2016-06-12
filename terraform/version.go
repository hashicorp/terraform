package terraform

import (
	"github.com/hashicorp/go-version"
)

// The main version number that is being run at the moment.
const Version = "0.7.0"

// A pre-release marker for the version. If this is "" (empty string)
// then it means that it is a final release. Otherwise, this is a pre-release
// such as "dev" (in development), "beta", "rc1", etc.
const VersionPrerelease = "dev"

// SemVersion is an instance of version.Version. This has the secondary
// benefit of verifying during tests and init time that our version is a
// proper semantic version, which should always be the case.
var SemVersion = version.Must(version.NewVersion(Version))

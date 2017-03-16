package scwversion

import "fmt"

var (
	// VERSION represents the semver version of the package
	VERSION = "v1.11.1+dev"

	// GITCOMMIT represents the git commit hash of the package, it is configured at build time
	GITCOMMIT string
)

// UserAgent returns a string to be used by API
func UserAgent() string {
	return fmt.Sprintf("scw/%v", VERSION)
}

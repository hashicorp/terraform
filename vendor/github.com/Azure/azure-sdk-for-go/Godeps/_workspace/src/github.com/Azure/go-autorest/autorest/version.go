package autorest

import (
	"fmt"
)

const (
	major        = "1"
	minor        = "1"
	patch        = "1"
	tag          = ""
	semVerFormat = "%s.%s.%s%s"
)

// Version returns the semantic version (see http://semver.org).
func Version() string {
	return fmt.Sprintf(semVerFormat, major, minor, patch, tag)
}

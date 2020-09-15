// +build !linux,!windows,!darwin,!aix,!dragonfly,!freebsd,!netbsd,!openbsd,!solaris

// The above build constraint must contain the negation of all of the build
// constraints found in the other app_*.go files, to catch any other OS
// we haven't accounted for.

package userdirs

import (
	"fmt"
	"runtime"
)

func supportedOS() bool {
	return false
}

func forApp(name string, vendor string, bundleID string) Dirs {
	panic(fmt.Sprintf("cannot determine user directories on OS %q", runtime.GOOS))
}

// +build !windows

package wrappedstreams

import (
	"os"
)

func initPlatform() {
	// The standard streams are passed in via extra file descriptors.
	wrappedStdin = os.NewFile(uintptr(3), "stdin")
	wrappedStdout = os.NewFile(uintptr(4), "stdout")
	wrappedStderr = os.NewFile(uintptr(5), "stderr")
}

// Package wrappedstreams provides access to the standard OS streams
// (stdin, stdout, stderr) even if wrapped under panicwrap.
package wrappedstreams

import (
	"os"

	"github.com/mitchellh/panicwrap"
)

// Stdin returns the true stdin of the process.
func Stdin() *os.File {
	stdin, _, _ := fds()
	return stdin
}

// Stdout returns the true stdout of the process.
func Stdout() *os.File {
	_, stdout, _ := fds()
	return stdout
}

// Stderr returns the true stderr of the process.
func Stderr() *os.File {
	_, _, stderr := fds()
	return stderr
}

func fds() (stdin, stdout, stderr *os.File) {
	stdin, stdout, stderr = os.Stdin, os.Stdout, os.Stderr
	if panicwrap.Wrapped(nil) {
		initPlatform()
		stdin, stdout, stderr = wrappedStdin, wrappedStdout, wrappedStderr
	}
	return
}

// These are the wrapped standard streams. These are set up by the
// platform specific code in initPlatform.
var (
	wrappedStdin  *os.File
	wrappedStdout *os.File
	wrappedStderr *os.File
)

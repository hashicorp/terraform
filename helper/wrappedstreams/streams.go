// Package wrappedstreams provides access to the standard OS streams
// (stdin, stdout, stderr) even if wrapped under panicwrap.
package wrappedstreams

import (
	"os"

	"github.com/mitchellh/panicwrap"
)

// Stdin returns the true stdin of the process.
func Stdin() *os.File {
	stdin := os.Stdin
	if panicwrap.Wrapped(nil) {
		stdin = wrappedStdin
	}

	return stdin
}

// Stdout returns the true stdout of the process.
func Stdout() *os.File {
	stdout := os.Stdout
	if panicwrap.Wrapped(nil) {
		stdout = wrappedStdout
	}

	return stdout
}

// Stderr returns the true stderr of the process.
func Stderr() *os.File {
	stderr := os.Stderr
	if panicwrap.Wrapped(nil) {
		stderr = wrappedStderr
	}

	return stderr
}

// These are the wrapped standard streams. These are setup by the
// platform specific code in initPlatform.
var (
	wrappedStdin  *os.File
	wrappedStdout *os.File
	wrappedStderr *os.File
)

func init() {
	// Initialize the platform-specific code
	initPlatform()
}

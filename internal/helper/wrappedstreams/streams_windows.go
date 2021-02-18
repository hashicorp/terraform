// +build windows

package wrappedstreams

import (
	"log"
	"os"
	"syscall"
)

func initPlatform() {
	wrappedStdin = openConsole("CONIN$", os.Stdin)
	wrappedStdout = openConsole("CONOUT$", os.Stdout)
	wrappedStderr = wrappedStdout
}

// openConsole opens a console handle, using a backup if it fails.
// This is used to get the exact console handle instead of the redirected
// handles from panicwrap.
func openConsole(name string, backup *os.File) *os.File {
	// Convert to UTF16
	path, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		log.Printf("[ERROR] wrappedstreams: %s", err)
		return backup
	}

	// Determine the share mode
	var shareMode uint32
	switch name {
	case "CONIN$":
		shareMode = syscall.FILE_SHARE_READ
	case "CONOUT$":
		shareMode = syscall.FILE_SHARE_WRITE
	}

	// Get the file
	h, err := syscall.CreateFile(
		path,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		shareMode,
		nil,
		syscall.OPEN_EXISTING,
		0, 0)
	if err != nil {
		log.Printf("[ERROR] wrappedstreams: %s", err)
		return backup
	}

	// Create the Go file
	return os.NewFile(uintptr(h), name)
}

// +build !windows

package panicwrap

import (
	"os"
	"syscall"
)

var WrapSignals []os.Signal = []os.Signal{os.Interrupt, syscall.SIGTERM}

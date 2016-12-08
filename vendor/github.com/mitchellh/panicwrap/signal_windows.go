// +build windows

package panicwrap

import (
	"os"
)

var WrapSignals []os.Signal = []os.Signal{os.Interrupt}

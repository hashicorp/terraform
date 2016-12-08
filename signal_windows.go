// +build windows

package main

import (
	"os"
)

var interruptSignals []os.Signal = []os.Signal{os.Interrupt}

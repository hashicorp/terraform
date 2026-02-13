// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows
// +build !windows

package main

import (
	"os"
	"syscall"
)

var ignoreSignals = []os.Signal{os.Interrupt}
var forwardSignals = []os.Signal{syscall.SIGTERM}
